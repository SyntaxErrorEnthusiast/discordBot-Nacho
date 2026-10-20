// internal/components/sheetsRefreshButton.go

package components

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"

	"github.com/agper/discordbot/internal/logger"
	"github.com/agper/discordbot/internal/services"
)

// SheetsRefreshButtonPrefix is the custom ID prefix used to identify Refresh button interactions.
const SheetsRefreshButtonPrefix = "sheets_refresh:"

// sheetState holds the original URL plus the spreadsheet ID and currently-selected GID
// for a rendered sheet message.
type sheetState struct {
	rawURL        string
	spreadsheetID string
	currentGID    int64
}

// sheetStateStore holds the in-memory mapping of short hash → sheet state.
// Entries live as long as the bot process, which is acceptable for a simple bot.
var (
	sheetStateStore      = make(map[string]*sheetState)
	sheetStateStoreMutex sync.RWMutex
)

// StoreSheetStateAndGetHash stores the given sheet state in memory and returns a short hash
// suitable for embedding in a Discord component custom ID.
func StoreSheetStateAndGetHash(rawURL string, spreadsheetID string, currentGID int64) string {
	// Include the GID in the hash so that switching tabs produces a distinct key.
	hasher := sha1.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%d", spreadsheetID, currentGID)))
	shortHash := fmt.Sprintf("%x", hasher.Sum(nil))[:16]

	sheetStateStoreMutex.Lock()
	sheetStateStore[shortHash] = &sheetState{
		rawURL:        rawURL,
		spreadsheetID: spreadsheetID,
		currentGID:    currentGID,
	}
	sheetStateStoreMutex.Unlock()

	return shortHash
}

// LookupSheetState retrieves stored sheet state by its hash.
func LookupSheetState(shortHash string) (rawURL string, spreadsheetID string, currentGID int64, found bool) {
	sheetStateStoreMutex.RLock()
	state, found := sheetStateStore[shortHash]
	sheetStateStoreMutex.RUnlock()
	if !found {
		return "", "", 0, false
	}
	return state.rawURL, state.spreadsheetID, state.currentGID, true
}

// BuildRefreshButton creates a Discord Refresh button for the given sheet state.
// The state is stored in memory keyed by a short hash embedded in the button's custom ID.
func BuildRefreshButton(rawURL string, spreadsheetID string, currentGID int64) discordgo.Button {
	shortHash := StoreSheetStateAndGetHash(rawURL, spreadsheetID, currentGID)
	return discordgo.Button{
		Label:    "Refresh",
		Emoji:    &discordgo.ComponentEmoji{Name: "🔄"},
		Style:    discordgo.SecondaryButton,
		CustomID: SheetsRefreshButtonPrefix + shortHash,
	}
}

// HandleSheetsRefreshButtonInteraction handles a click on the Refresh button.
// It looks up the stored state from the hash, re-fetches the sheet, and edits the original message.
func HandleSheetsRefreshButtonInteraction(
	discordSession *discordgo.Session,
	interaction *discordgo.InteractionCreate,
	sheetsAPIKey string,
) {
	shortHash := interaction.MessageComponentData().CustomID[len(SheetsRefreshButtonPrefix):]

	rawURL, spreadsheetID, currentGID, found := LookupSheetState(shortHash)
	if !found {
		_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button has expired. Please run `/sheets` again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	logContext := &logger.LogContext{
		Command: "sheets_refresh",
		UserID:  interaction.Member.User.ID,
		GuildID: interaction.GuildID,
	}

	// Acknowledge the interaction immediately so Discord doesn't show a timeout.
	_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	renderedPNGBytes, screenshotError := services.ScreenshotSheetByGID(context.Background(), spreadsheetID, currentGID, sheetsAPIKey)
	if screenshotError != nil {
		var errorMessage string
		if errors.Is(screenshotError, services.ErrSheetNotPublic) {
			errorMessage = `That sheet doesn't appear to be public. Make sure sharing is set to "Anyone with the link".`
		} else {
			logger.Error("Failed to refresh sheet screenshot", logContext, screenshotError)
			errorMessage = "Something went wrong while refreshing the screenshot. Please try again."
		}
		_, _ = discordSession.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
			Content: &errorMessage,
			Files:   []*discordgo.File{},
		})
		return
	}

	// Re-fetch the tab list to rebuild the select menu (tabs may have been added/removed).
	sheetList, listError := services.ListSheets(context.Background(), spreadsheetID, sheetsAPIKey)
	if listError != nil {
		logger.Error("Failed to list sheets on refresh", logContext, listError)
	}

	newHash := StoreSheetStateAndGetHash(rawURL, spreadsheetID, currentGID)
	refreshButton := BuildRefreshButton(rawURL, spreadsheetID, currentGID)

	// The refresh button and the select menu must be in separate ActionsRows —
	// Discord requires a select menu to be the only component in its row.
	updatedComponents := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{refreshButton}},
	}
	if len(sheetList) > 1 {
		updatedComponents = append(updatedComponents, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				BuildTabSelectMenu(spreadsheetID, sheetList, currentGID, newHash),
			},
		})
	}

	components := updatedComponents

	_, _ = discordSession.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
		// Pass an empty Attachments slice to remove all existing attachments before
		// adding the new file. Without this Discord appends rather than replaces.
		Attachments: &[]*discordgo.MessageAttachment{},
		Files: []*discordgo.File{
			{Name: "sheet.png", Reader: bytes.NewReader(renderedPNGBytes)},
		},
		Components: &components,
	})
}
