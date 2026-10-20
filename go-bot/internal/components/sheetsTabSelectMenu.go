// internal/components/sheetsTabSelectMenu.go
// Select menu component for switching between tabs in a Google Sheets message.

package components

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/agper/discordbot/internal/logger"
	"github.com/agper/discordbot/internal/services"
)

// SheetsTabSelectPrefix is the custom ID prefix used to identify tab select menu interactions.
// The suffix is the state hash used to look up the spreadsheet ID.
const SheetsTabSelectPrefix = "sheets_tab:"

// BuildTabSelectMenu creates a Discord select menu listing all tabs in a spreadsheet.
// The currently-active tab is marked with a checkmark in its description.
// stateHash is the hash already registered in the state store for this message,
// so the menu shares the same lookup key as the refresh button.
func BuildTabSelectMenu(
	spreadsheetID string,
	sheetList []services.SheetInfo,
	currentGID int64,
	stateHash string,
) discordgo.SelectMenu {
	// Discord caps select menu options at 25.
	const maxOptions = 25

	options := make([]discordgo.SelectMenuOption, 0, len(sheetList))
	for index, sheetInfo := range sheetList {
		if index >= maxOptions {
			break
		}

		isCurrentTab := sheetInfo.GID == currentGID
		description := ""
		if isCurrentTab {
			description = "Currently viewing"
		}
		title := sanitizeTitle(sheetInfo.Title, currentGID)
		logContext := &logger.LogContext{
			Command: "BuildTabSelectMenu",
		}
		logger.Error("New Name:", logContext, fmt.Errorf("%s", title))
		options = append(options, discordgo.SelectMenuOption{
			Label:       title,
			Value:       strconv.FormatInt(sheetInfo.GID, 10),
			Description: description,
			Default:     isCurrentTab,
		})
	}

	placeholder := "Switch tab"
	if len(sheetList) > maxOptions {
		placeholder = fmt.Sprintf("Switch tab (showing first %d of %d)", maxOptions, len(sheetList))
	}

	return discordgo.SelectMenu{
		MenuType:    discordgo.StringSelectMenu,
		CustomID:    SheetsTabSelectPrefix + stateHash,
		Placeholder: placeholder,
		Options:     options,
	}
}

// HandleTabSelectInteraction handles a selection from the tab select menu.
// It looks up the stored sheet state, re-renders the chosen tab, and edits the message in place.
func HandleTabSelectInteraction(discordSession *discordgo.Session, interaction *discordgo.InteractionCreate, sheetsAPIKey string) {
	componentData := interaction.MessageComponentData()
	stateHash := strings.TrimPrefix(componentData.CustomID, SheetsTabSelectPrefix)

	if len(componentData.Values) == 0 {
		// No value selected — nothing to do.
		_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
		return
	}

	selectedGID, parseError := strconv.ParseInt(componentData.Values[0], 10, 64)
	if parseError != nil {
		_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid tab selection. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	rawURL, spreadsheetID, _, found := LookupSheetState(stateHash)
	if !found {
		_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This menu has expired. Please run `/sheets` again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	logContext := &logger.LogContext{
		Command: "sheets_tab_select",
		UserID:  interaction.Member.User.ID,
		GuildID: interaction.GuildID,
	}

	// Acknowledge immediately — rendering takes a few seconds.
	_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	renderedPNGBytes, screenshotError := services.ScreenshotSheetByGID(context.Background(), spreadsheetID, selectedGID, sheetsAPIKey)
	if screenshotError != nil {
		var errorMessage string
		if errors.Is(screenshotError, services.ErrSheetNotPublic) {
			errorMessage = `That sheet doesn't appear to be public. Make sure sharing is set to "Anyone with the link".`
		} else {
			logger.Error("Failed to screenshot selected tab", logContext, screenshotError)
			errorMessage = "Something went wrong while switching tabs. Please try again."
		}
		_, _ = discordSession.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
			Content: &errorMessage,
		})
		return
	}

	// Fetch the latest tab list so the menu reflects any structural changes.
	sheetList, listError := services.ListSheets(context.Background(), spreadsheetID, sheetsAPIKey)
	if listError != nil {
		logger.Error("Failed to list sheets after tab switch", logContext, listError)
	}

	// Register a new state entry for the newly-selected tab so the refresh button
	// and the select menu both target the correct tab going forward.
	newHash := StoreSheetStateAndGetHash(rawURL, spreadsheetID, selectedGID)
	refreshButton := BuildRefreshButton(rawURL, spreadsheetID, selectedGID)

	// The refresh button and the select menu must be in separate ActionsRows —
	// Discord requires a select menu to be the only component in its row.
	updatedComponents := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{refreshButton}},
	}
	if len(sheetList) > 1 {
		updatedComponents = append(updatedComponents, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				BuildTabSelectMenu(spreadsheetID, sheetList, selectedGID, newHash),
			},
		})
	}

	_, _ = discordSession.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
		Attachments: &[]*discordgo.MessageAttachment{},
		Files: []*discordgo.File{
			{Name: "sheet.png", Reader: bytes.NewReader(renderedPNGBytes)},
		},
		Components: &updatedComponents,
	})
}

// sanitizeTitle ensures the title is between 0 and 100 (to abide by discords title limit)
func sanitizeTitle(title string, gID int64) string {
	const lengthLimit = 100
	title = strings.TrimSpace(title)
	if title == "" {
		return strconv.FormatInt(gID, 10)
	}
	if len(title) > lengthLimit {
		return title[:lengthLimit-3] + "..."
	}

	return title
}
