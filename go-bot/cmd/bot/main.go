// cmd/bot/main.go
// Entry point — loads configuration, wires up the Discord client, and routes interactions.

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"github.com/agper/discordbot/internal/commands"
	"github.com/agper/discordbot/internal/components"
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

	discordSession.Identify.Intents = discordgo.IntentsGuilds

	discordSession.AddHandler(func(session *discordgo.Session, readyEvent *discordgo.Ready) {
		logger.Info("Logged in as "+readyEvent.User.String(), nil)
	})

	discordSession.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		switch interaction.Type {
		case discordgo.InteractionApplicationCommand:
			commandName := interaction.ApplicationCommandData().Name
			switch commandName {
			case "sheets":
				commands.HandleSheetsCommand(session, interaction, applicationConfig.SheetsKey)
			default:
				logContext := &logger.LogContext{
					Command: commandName,
					UserID:  interaction.Member.User.ID,
					GuildID: interaction.GuildID,
				}
				logger.Warn("Received unknown command", logContext)
				_ = session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Unknown command.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}

		case discordgo.InteractionMessageComponent:
			customID := interaction.MessageComponentData().CustomID
			if len(customID) >= len(components.SheetsRefreshButtonPrefix) &&
				customID[:len(components.SheetsRefreshButtonPrefix)] == components.SheetsRefreshButtonPrefix {
				components.HandleSheetsRefreshButtonInteraction(session, interaction, applicationConfig.SheetsKey)
			}
		}
	})

	if connectionError := discordSession.Open(); connectionError != nil {
		logger.Error("Failed to open Discord WebSocket connection", nil, connectionError)
		os.Exit(1)
	}
	defer discordSession.Close()

	logger.Info("Bot is running. Press CTRL-C to exit.", nil)

	shutdownSignalChannel := make(chan os.Signal, 1)
	signal.Notify(shutdownSignalChannel, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignalChannel

	logger.Info("Shutdown signal received. Stopping bot.", nil)
}
