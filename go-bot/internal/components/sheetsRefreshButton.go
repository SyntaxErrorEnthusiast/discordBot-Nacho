// internal/components/sheetsRefreshButton.go

package components

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"

	"github.com/agper/discordbot/internal/logger"
	"github.com/agper/discordbot/internal/services"
)

// SheetsRefreshButtonPrefix is the custom ID prefix used to identify Refresh button interactions.
const SheetsRefreshButtonPrefix = "sheets_refresh:"

// sheetURLStore holds the in-memory mapping of short hash → original Google Sheets URL.
// Entries live as long as the bot process, which is acceptable for a simple bot.
var (
	sheetURLStore      = make(map[string]string)
	sheetURLStoreMutex sync.RWMutex
)

// storeSheetURLAndGetHash stores the given URL in memory and returns a short hash
// suitable for embedding in a Discord button custom ID.
func storeSheetURLAndGetHash(rawSheetURL string) string {
	hasher := sha1.New()
	hasher.Write([]byte(rawSheetURL))
	shortHash := fmt.Sprintf("%x", hasher.Sum(nil))[:16]

	sheetURLStoreMutex.Lock()
	sheetURLStore[shortHash] = rawSheetURL
	sheetURLStoreMutex.Unlock()

	return shortHash
}

// BuildRefreshButton creates a Discord Refresh button for the given Google Sheets URL.
// The URL is stored in memory keyed by a short hash embedded in the button's custom ID.
func BuildRefreshButton(rawSheetURL string) discordgo.Button {
	shortHash := storeSheetURLAndGetHash(rawSheetURL)
	return discordgo.Button{
		Label:    "Refresh",
		Emoji:    &discordgo.ComponentEmoji{Name: "🔄"},
		Style:    discordgo.SecondaryButton,
		CustomID: SheetsRefreshButtonPrefix + shortHash,
	}
}

// HandleSheetsRefreshButtonInteraction handles a click on the Refresh button.
// It looks up the original URL from the hash, re-fetches the sheet, and edits the original message.
func HandleSheetsRefreshButtonInteraction(
	discordSession *discordgo.Session,
	interaction *discordgo.InteractionCreate,
	sheetsAPIKey string,
) {
	shortHash := interaction.MessageComponentData().CustomID[len(SheetsRefreshButtonPrefix):]

	sheetURLStoreMutex.RLock()
	rawSheetURL, urlFound := sheetURLStore[shortHash]
	sheetURLStoreMutex.RUnlock()

	if !urlFound {
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

	renderedPNGBytes, screenshotError := services.ScreenshotSheet(context.Background(), rawSheetURL, sheetsAPIKey)
	if screenshotError != nil {
		var errorMessage string
		if screenshotError == services.ErrSheetNotPublic {
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

	_, _ = discordSession.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
		// Pass an empty Attachments slice to remove all existing attachments before
		// adding the new file. Without this Discord appends rather than replaces.
		Attachments: &[]*discordgo.MessageAttachment{},
		Files: []*discordgo.File{
			{Name: "sheet.png", Reader: bytes.NewReader(renderedPNGBytes)},
		},
	})
}
