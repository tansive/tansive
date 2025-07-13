package session

import (
	"context"
	"path/filepath"

	jsonitor "github.com/json-iterator/go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/session/hashlog"
)

// auditLogInfo contains audit logging configuration and state.
// Manages audit logger instances, completion channels, and signing keys.
type auditLogInfo struct {
	auditLogger      zerolog.Logger
	auditLogComplete chan string
	auditLogPubKey   []byte
	auditLogCancel   context.CancelFunc
}

// GetAuditLogPath generates the file path for a session's audit log.
// Returns the full path to the audit log file in the configured audit log directory.
func GetAuditLogPath(sessionID string) string {
	// get os application data directory
	auditLogDir := config.GetAuditLogDir()
	auditLogPath := filepath.Join(auditLogDir, sessionID+".tlog")
	return auditLogPath
}

// InitAuditLog initializes audit logging for a session.
// Creates a signed hash log writer and starts background processing of audit events.
// Returns an error if initialization fails.
func InitAuditLog(ctx context.Context, session *session) apperrors.Error {
	auditLogPath := GetAuditLogPath(session.id.String())
	log.Ctx(ctx).Info().Str("audit_log_path", auditLogPath).Msg("initializing audit log")
	session.auditLogInfo.auditLogger = session.getLogger(TopicAuditLog).With().Str("actor", "system").Logger()
	session.auditLogInfo.auditLogPubKey = config.GetRuntimeConfig().LogSigningKey.PublicKey
	session.auditLogInfo.auditLogComplete = make(chan string, 1)

	logWriter, err := hashlog.NewHashLogWriter(auditLogPath, 100, config.GetRuntimeConfig().LogSigningKey.PrivateKey)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create audit logger")
		session.auditLogInfo.auditLogComplete <- ""
		return ErrSessionError.Msg("failed to create audit logger")
	}
	auditLog, unsubAuditLog := GetEventBus().Subscribe(session.getTopic(TopicAuditLog), 100)

	finalizeLog := func() {
		session.auditLogInfo.auditLogger.Info().
			Str("tangent_id", config.GetRuntimeConfig().TangentID.String()).
			Str("tangent_url", config.GetURL()).
			Str("event", "log_finalize").
			Msg("log finalized")
		logWriter.Flush()
		logWriter.Close()
		unsubAuditLog()
		session.auditLogInfo.auditLogComplete <- auditLogPath
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Ctx(ctx).Error().Msgf("panic in audit log: %v", r)
			}
			finalizeLog()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-auditLog:
				if !ok {
					return
				}
				data, ok := event.Data.([]byte)
				if !ok {
					continue
				}
				var logMap map[string]any
				if err := jsonitor.Unmarshal(data, &logMap); err != nil {
					log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal audit log")
					continue
				}
				logWriter.AddEntry(logMap)
			}
		}
	}()
	session.auditLogInfo.auditLogger.Info().
		Str("tangent_id", config.GetRuntimeConfig().TangentID.String()).
		Str("tangent_url", config.GetURL()).
		Str("event", "log_start").
		Msg("log started")
	return nil
}
