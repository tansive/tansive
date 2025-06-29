package tangentcommon

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// DefaultConfigFile is the default configuration file name.
// Used when no specific configuration file is provided.
const DefaultConfigFile = "config.toml"

// InitLogger initializes the global logger with Unix timestamp format.
// Configures zerolog to use Unix millisecond timestamps for consistent logging.
func InitLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
}
