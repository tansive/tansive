// Package logtrace provides logging and tracing utilities for the application.
// It integrates with zerolog for structured logging and supports request tracing.
package logtrace

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger initializes the global logger with Unix timestamp format.
// Configures zerolog to output to stderr with timestamps.
func InitLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
}
