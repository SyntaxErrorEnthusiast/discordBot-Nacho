package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type ApplicationConfig struct {
	BotToken  string
	SheetsKey string
	ClientID  string
	// GuildID is optional — empty means commands are deployed globally.
	GuildID string
}

// LoadApplicationConfig reads the .env file (if present) and validates all required variables.
// Returns an error if any required variable is missing or empty.
func LoadApplicationConfig() (*ApplicationConfig, error) {
	// .env is optional in production — env vars may be injected by Docker or the host.
	_ = godotenv.Load()

	applicationConfig := &ApplicationConfig{
		GuildID: os.Getenv("GUILD_ID"),
	}

	var missingVariables []string
	applicationConfig.BotToken = requireEnvironmentVariable("BOT_TOKEN", &missingVariables)
	applicationConfig.SheetsKey = requireEnvironmentVariable("SHEETS_KEY", &missingVariables)
	applicationConfig.ClientID = requireEnvironmentVariable("CLIENT_ID", &missingVariables)

	if len(missingVariables) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVariables)
	}

	return applicationConfig, nil
}

// requireEnvironmentVariable reads an environment variable and appends its name
// to missingVariables if it is empty.
func requireEnvironmentVariable(variableName string, missingVariables *[]string) string {
	variableValue := os.Getenv(variableName)
	if variableValue == "" {
		*missingVariables = append(*missingVariables, variableName)
	}
	return variableValue
}
