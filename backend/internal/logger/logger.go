package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New creates a new zerolog logger instance
func New() zerolog.Logger {
	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339

	// Pretty print for development
	if os.Getenv("APP_ENV") == "development" {
		return zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		}).With().Timestamp().Caller().Logger()
	}

	// JSON output for production
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
