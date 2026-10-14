// cmd/deploy/main.go
// One-shot slash command registration tool — equivalent to deploy-commands.ts.
// Uses a bulk overwrite (PUT) so stale commands are removed automatically.
// Run once after adding or modifying slash commands.
//
// Usage:
//   go run ./cmd/deploy

package main

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"

	"github.com/agper/discordbot/internal/config"
	"github.com/agper/discordbot/internal/logger"
)

func main() {
	applicationConfig, configLoadError := config.LoadApplicationConfig()
	if configLoadError != nil {
		logger.Error("Failed to load application configuration", nil, configLoadError)
		os.Exit(1)
	}

	discordSession, sessionCreateError := discordgo.New("Bot " + applicationConfig.BotToken)
	if sessionCreateError != nil {
		logger.Error("Failed to create Discord session", nil, sessionCreateError)
		os.Exit(1)
	}

	// The full list of commands the bot should have. Any command not in this
	// list will be removed from Discord — this is a full replace, not an append.
	desiredCommands := []*discordgo.ApplicationCommand{
		{
			Name:        "sheets",
			Description: "Screenshots a public Google Sheet and posts it here.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "url",
					Description: "The public Google Sheets URL",
					Required:    true,
				},
			},
		},
	}

	fmt.Println("Deploying slash commands (full replace)...")

	// BulkOverwriteGuildApplicationCommands / BulkOverwriteApplicationCommands
	// both use PUT under the hood — identical to the TS version's rest.put().
	// This guarantees stale commands are wiped and only desiredCommands remain.
	if applicationConfig.GuildID != "" {
		_, overwriteError := discordSession.ApplicationCommandBulkOverwrite(
			applicationConfig.ClientID,
			applicationConfig.GuildID,
			desiredCommands,
		)
		if overwriteError != nil {
			logger.Error("Failed to bulk overwrite guild commands", nil, overwriteError)
			os.Exit(1)
		}
		fmt.Printf("Commands deployed to guild %s.\n", applicationConfig.GuildID)
	} else {
		_, overwriteError := discordSession.ApplicationCommandBulkOverwrite(
			applicationConfig.ClientID,
			"",
			desiredCommands,
		)
		if overwriteError != nil {
			logger.Error("Failed to bulk overwrite global commands", nil, overwriteError)
			os.Exit(1)
		}
		fmt.Println("Commands deployed globally.")
	}

	fmt.Println("Successfully reloaded application (/) commands.")
}
