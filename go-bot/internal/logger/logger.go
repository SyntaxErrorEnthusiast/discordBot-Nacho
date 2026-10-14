// internal/logger/logger.go
// Centralized structured logger. All packages use this instead of log/fmt directly.
// Outputs JSON lines to stdout so logs are machine-readable in production.

package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogContext carries optional Discord-specific fields attached to a log entry.
type LogContext struct {
	Command string `json:"command,omitempty"`
	UserID  string `json:"userId,omitempty"`
	GuildID string `json:"guildId,omitempty"`
}

// LogEntry is the structured JSON object written to stdout for each log call.
type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	Level     LogLevel `json:"level"`
	Message   string   `json:"message"`
	Command   string   `json:"command,omitempty"`
	UserID    string   `json:"userId,omitempty"`
	GuildID   string   `json:"guildId,omitempty"`
	Error     string   `json:"error,omitempty"`
	Stack     string   `json:"stack,omitempty"`
}

// writeLogEntry serialises a LogEntry to JSON and writes it to stdout.
func writeLogEntry(logLevel LogLevel, message string, logContext *LogContext, logError error) {
	logEntry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     logLevel,
		Message:   message,
	}

	if logContext != nil {
		logEntry.Command = logContext.Command
		logEntry.UserID = logContext.UserID
		logEntry.GuildID = logContext.GuildID
	}

	if logError != nil {
		logEntry.Error = logError.Error()
		// Go doesn't attach stack traces to errors by default;
		// use %+v to capture the full wrapped error chain.
		logEntry.Stack = fmt.Sprintf("%+v", logError)
	}

	jsonBytes, marshalError := json.Marshal(logEntry)
	if marshalError != nil {
		// Fallback — should never happen with a well-formed LogEntry
		fmt.Fprintf(os.Stdout, `{"level":"error","message":"failed to marshal log entry"}`+"\n")
		return
	}

	fmt.Fprintf(os.Stdout, "%s\n", jsonBytes)
}

// Info logs an informational message with optional context.
func Info(message string, logContext *LogContext) {
	writeLogEntry(LogLevelInfo, message, logContext, nil)
}

// Warn logs a warning message with optional context.
func Warn(message string, logContext *LogContext) {
	writeLogEntry(LogLevelWarn, message, logContext, nil)
}

// Error logs an error message with optional context and error value.
func Error(message string, logContext *LogContext, logError error) {
	writeLogEntry(LogLevelError, message, logContext, logError)
}
