package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// New creates a new zerolog logger instance.
// Log level is read from LOG_LEVEL env var. Defaults to "debug" in development, "info" in production.
func New() zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	// Determine log level from environment
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		if os.Getenv("APP_ENV") == "production" {
			level = "info"
		} else {
			level = "debug"
		}
	}

	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

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
