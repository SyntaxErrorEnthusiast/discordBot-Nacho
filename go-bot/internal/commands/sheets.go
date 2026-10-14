// internal/commands/sheets.go

package commands

import (
	"bytes"
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"

	"github.com/agper/discordbot/internal/components"
	"github.com/agper/discordbot/internal/logger"
	"github.com/agper/discordbot/internal/services"
)

// HandleSheetsCommand handles the /sheets slash command interaction.
// It defers the reply, fetches and renders the Google Sheet, and posts the result.
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

	// Defer the reply immediately — the export and render pipeline takes several seconds.
	_ = discordSession.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	logContext := &logger.LogContext{
		Command: "sheets",
		UserID:  interaction.Member.User.ID,
		GuildID: interaction.GuildID,
	}

	renderedPNGBytes, screenshotError := services.ScreenshotSheet(context.Background(), rawSheetURL, sheetsAPIKey)
	if screenshotError != nil {
		var errorMessage string
		if errors.Is(screenshotError, services.ErrSheetNotPublic) {
			errorMessage = `That sheet doesn't appear to be public. Make sure sharing is set to "Anyone with the link".`
		} else {
			logger.Error("Failed to screenshot sheet", logContext, screenshotError)
			errorMessage = "Something went wrong while generating the screenshot. Please try again."
		}
		_, _ = discordSession.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
			Content: errorMessage,
		})
		return
	}

	refreshButton := components.BuildRefreshButton(rawSheetURL)
	_, _ = discordSession.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
		Files: []*discordgo.File{
			{Name: "sheet.png", Reader: bytes.NewReader(renderedPNGBytes)},
		},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{refreshButton},
			},
		},
	})
}
