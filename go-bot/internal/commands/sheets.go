// internal/commands/sheets.go

package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/agper/discordbot/internal/components"
	"github.com/agper/discordbot/internal/logger"
	"github.com/agper/discordbot/internal/services"
)

// HandleSheetsCommand handles the /sheets slash command interaction.
// It defers the reply, fetches and renders the Google Sheet, and posts the result
// with a refresh button and (when multiple tabs exist) a tab select menu.
func HandleSheetsCommand(
	discordSession *discordgo.Session,
	interaction *discordgo.InteractionCreate,
	sheetsAPIKey string,
) {
	commandData := interaction.ApplicationCommandData()

	var rawSheetURL string
	for _, commandOption := range commandData.Options {
		if commandOption.Name == "url" {
			rawSheetURL = commandOption.StringValue()
			break
		}
	}

	if rawSheetURL == "" {
		_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Please provide a Google Sheets URL.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	spreadsheetID, sheetGID, parseError := services.ParseSheetURL(rawSheetURL)
	if parseError != nil {
		_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "That doesn't look like a valid Google Sheets URL.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Defer the reply immediately — the export and render pipeline takes several seconds.
	_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	logContext := &logger.LogContext{
		Command: "sheets",
		UserID:  interaction.Member.User.ID,
		GuildID: interaction.GuildID,
	}

	// Fetch the tab list and render the initial screenshot concurrently — they are independent.
	type screenshotResult struct {
		pngBytes []byte
		err      error
	}
	type listResult struct {
		sheets []services.SheetInfo
		err    error
	}

	screenshotChannel := make(chan screenshotResult, 1)
	listChannel := make(chan listResult, 1)

	go func() {
		pngBytes, err := services.ScreenshotSheetByGID(context.Background(), spreadsheetID, sheetGID, sheetsAPIKey)
		screenshotChannel <- screenshotResult{pngBytes, err}
	}()

	go func() {
		sheetList, err := services.ListSheets(context.Background(), spreadsheetID, sheetsAPIKey)
		listChannel <- listResult{sheetList, err}
	}()

	screenshotRes := <-screenshotChannel
	if screenshotRes.err != nil {
		var errorMessage string
		if errors.Is(screenshotRes.err, services.ErrSheetNotPublic) {
			errorMessage = `That sheet doesn't appear to be public. Make sure sharing is set to "Anyone with the link".`
		} else {
			logger.Error("Failed to screenshot sheet", logContext, screenshotRes.err)
			errorMessage = "Something went wrong while generating the screenshot. Please try again."
		}
		_, _ = discordSession.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
			Content: errorMessage,
		})
		return
	}

	listRes := <-listChannel
	if listRes.err != nil {
		logger.Error("Failed to list sheets", logContext, listRes.err)
	}

	stateHash := components.StoreSheetStateAndGetHash(rawSheetURL, spreadsheetID, sheetGID)
	refreshButton := components.BuildRefreshButton(rawSheetURL, spreadsheetID, sheetGID)

	// The refresh button and the select menu must be in separate ActionsRows —
	// Discord requires a select menu to be the only component in its row.
	messageComponents := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{refreshButton},
		},
	}
	if len(listRes.sheets) > 1 {
		logger.Info(fmt.Sprintf("Sheet has %d tabs, adding select menu", len(listRes.sheets)), logContext)
		messageComponents = append(messageComponents, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				components.BuildTabSelectMenu(spreadsheetID, listRes.sheets, sheetGID, stateHash),
			},
		})
	}

	_, followupErr := discordSession.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
		Files: []*discordgo.File{
			{Name: "sheet.png", Reader: bytes.NewReader(screenshotRes.pngBytes)},
		},
		Components: messageComponents,
	})
	if followupErr != nil {
		logger.Error("Failed to send followup message", logContext, followupErr)
	}
}
