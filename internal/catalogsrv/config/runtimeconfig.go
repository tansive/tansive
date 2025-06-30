package config

import (
	"os"

	"github.com/rs/zerolog/log"
)

func Init() {
	log.Info().Msg("Initializing runtime config")
	auditLogPath := Config().AuditLog.GetPath()
	log.Info().Msgf("Audit log path: %s", auditLogPath)
	err := os.MkdirAll(auditLogPath, 0755)
	if err != nil {
		log.Error().Err(err).Msg("Error creating audit log path")
		os.Exit(1)
	}
}
