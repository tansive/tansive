package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

// UpsertSession creates a new session in the database.
func (mm *metadataManager) UpsertSession(ctx context.Context, session *models.Session) (err apperrors.Error) {
	tx, errStd := mm.conn().BeginTx(ctx, nil)
	if errStd != nil {
		log.Ctx(ctx).Error().Err(errStd).Msg("failed to begin transaction")
		return dberror.ErrDatabase.Err(errStd)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Ctx(ctx).Error().Err(rollbackErr).Msg("failed to rollback transaction")
			}
		}
	}()

	err = mm.upsertSessionWithTransaction(ctx, session, tx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create session")
		return err
	}

	if errStd := tx.Commit(); errStd != nil {
		log.Ctx(ctx).Error().Err(errStd).Msg("failed to commit transaction")
		return dberror.ErrDatabase.Err(errStd)
	}

	return nil
}

// upsertSessionWithTransaction handles the actual session creation within a transaction.
func (mm *metadataManager) upsertSessionWithTransaction(ctx context.Context, session *models.Session, tx *sql.Tx) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	session.TenantID = tenantID

	query := `
		INSERT INTO sessions (
			session_id, skillset, skill, view_id, 
			tangent_id, status_summary, status, info, user_id, catalog_id, 
			variant_id, tenant_id, started_at, ended_at, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (tenant_id, session_id) DO UPDATE SET
			skillset = EXCLUDED.skillset,
			skill = EXCLUDED.skill,
			view_id = EXCLUDED.view_id,
			tangent_id = EXCLUDED.tangent_id,
			status_summary = EXCLUDED.status_summary,
			status = EXCLUDED.status,
			info = EXCLUDED.info,
			user_id = EXCLUDED.user_id,
			catalog_id = EXCLUDED.catalog_id,
			variant_id = EXCLUDED.variant_id,
			started_at = EXCLUDED.started_at,
			ended_at = EXCLUDED.ended_at,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
		RETURNING session_id
	`

	err := tx.QueryRowContext(ctx, query,
		session.SessionID,
		session.SkillSet,
		session.Skill,
		session.ViewID,
		session.TangentID,
		session.StatusSummary,
		session.Status,
		session.Info,
		session.UserID,
		session.CatalogID,
		session.VariantID,
		session.TenantID,
		session.StartedAt,
		session.EndedAt,
		session.ExpiresAt,
	).Scan(&session.SessionID)

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to insert/update session")
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}

// GetSession retrieves a session by its ID.
func (mm *metadataManager) GetSession(ctx context.Context, sessionID uuid.UUID) (*models.Session, apperrors.Error) {
	log.Ctx(ctx).Info().Msgf("Getting session %s", sessionID.String())
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT 
			s.session_id,
			s.skillset,
			s.skill,
			s.view_id,
			s.tangent_id,
			s.status_summary,
			s.status,
			s.info,
			s.user_id,
			s.catalog_id,
			s.variant_id,
			s.tenant_id,
			s.created_at,
			s.started_at,
			s.ended_at,
			s.updated_at,
			s.expires_at
		FROM 
			sessions s
		WHERE 
			s.tenant_id = $1 AND 
			s.session_id = $2;
	`

	var session models.Session
	err := mm.conn().QueryRowContext(ctx, query, tenantID, sessionID).
		Scan(
			&session.SessionID,
			&session.SkillSet,
			&session.Skill,
			&session.ViewID,
			&session.TangentID,
			&session.StatusSummary,
			&session.Status,
			&session.Info,
			&session.UserID,
			&session.CatalogID,
			&session.VariantID,
			&session.TenantID,
			&session.CreatedAt,
			&session.StartedAt,
			&session.EndedAt,
			&session.UpdatedAt,
			&session.ExpiresAt,
		)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("session not found")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}

	return &session, nil
}

// UpdateSessionStatus updates the status of a session.
func (mm *metadataManager) UpdateSessionStatus(ctx context.Context, sessionID uuid.UUID, statusSummary string, status json.RawMessage) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	query := `
		UPDATE sessions
		SET 
			status_summary = $3,
			status = $4,
			updated_at = NOW()
		WHERE tenant_id = $1 AND session_id = $2
	`

	result, err := mm.conn().ExecContext(ctx, query,
		tenantID,
		sessionID,
		statusSummary,
		status,
	)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update session status")
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("session not found")
	}

	return nil
}

// UpdateSessionEnd marks a session as ended and updates its status.
func (mm *metadataManager) UpdateSessionEnd(ctx context.Context, sessionID uuid.UUID, statusSummary string, status json.RawMessage) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	query := `
		UPDATE sessions
		SET 
			status_summary = $3,
			status = $4,
			ended_at = NOW(),
			updated_at = NOW()
		WHERE tenant_id = $1 AND session_id = $2
	`

	result, err := mm.conn().ExecContext(ctx, query,
		tenantID,
		sessionID,
		statusSummary,
		status,
	)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update session end")
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("session not found")
	}

	return nil
}

// UpdateSessionInfo updates the info field of a session.
func (mm *metadataManager) UpdateSessionInfo(ctx context.Context, sessionID uuid.UUID, info json.RawMessage) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	query := `
		UPDATE sessions
		SET 
			info = $3,
			updated_at = NOW()
		WHERE tenant_id = $1 AND session_id = $2
	`

	result, err := mm.conn().ExecContext(ctx, query,
		tenantID,
		sessionID,
		info,
	)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update session info")
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("session not found")
	}

	return nil
}

// DeleteSession deletes a session by its ID.
func (mm *metadataManager) DeleteSession(ctx context.Context, sessionID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	query := `
		DELETE FROM sessions
		WHERE tenant_id = $1 AND session_id = $2
	`

	result, err := mm.conn().ExecContext(ctx, query, tenantID, sessionID)
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("session not found")
	}

	return nil
}

// ListSessionsByCatalog retrieves all sessions for a specific catalog.
// Sessions are ordered by creation time in descending order (newest first).
func (mm *metadataManager) ListSessionsByCatalog(ctx context.Context, catalogID uuid.UUID) ([]*models.Session, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT 
			session_id, skillset, skill, view_id,
			tangent_id, status_summary, status, info, user_id, catalog_id,
			variant_id, tenant_id, created_at, started_at,
			ended_at, updated_at, expires_at
		FROM sessions
		WHERE tenant_id = $1 AND catalog_id = $2
		ORDER BY created_at DESC
	`

	rows, err := mm.conn().QueryContext(ctx, query, tenantID, catalogID)
	if err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}
	defer rows.Close()

	var result []*models.Session

	for rows.Next() {
		var session models.Session
		err := rows.Scan(
			&session.SessionID,
			&session.SkillSet,
			&session.Skill,
			&session.ViewID,
			&session.TangentID,
			&session.StatusSummary,
			&session.Status,
			&session.Info,
			&session.UserID,
			&session.CatalogID,
			&session.VariantID,
			&session.TenantID,
			&session.CreatedAt,
			&session.StartedAt,
			&session.EndedAt,
			&session.UpdatedAt,
			&session.ExpiresAt,
		)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to scan session row")
			return nil, dberror.ErrDatabase.Err(err)
		}
		result = append(result, &session)
	}

	if err := rows.Err(); err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}

	return result, nil
}
