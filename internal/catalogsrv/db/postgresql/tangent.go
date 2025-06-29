package postgresql

import (
	"context"
	"database/sql"

	"github.com/jackc/pgconn"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

func (mm *metadataManager) CreateTangent(ctx context.Context, tangent *models.Tangent) (err apperrors.Error) {
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

	err = mm.createTangentWithTransaction(ctx, tangent, tx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create tangent")
		return err
	}

	if errStd := tx.Commit(); errStd != nil {
		log.Ctx(ctx).Error().Err(errStd).Msg("failed to commit transaction")
		return dberror.ErrDatabase.Err(errStd)
	}

	return nil
}

func (mm *metadataManager) createTangentWithTransaction(ctx context.Context, tangent *models.Tangent, tx *sql.Tx) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	tangent.TenantID = string(tenantID)

	if tangent.ID == uuid.Nil {
		return dberror.ErrInvalidInput.Msg("tangent id is required")
	}

	query := `
		INSERT INTO tangents (id, info, public_key, status, tenant_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := tx.QueryRowContext(ctx, query,
		tangent.ID,
		tangent.Info,
		tangent.PublicKey,
		tangent.Status,
		tangent.TenantID,
	).Scan(&tangent.ID)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch {
			case pgErr.Code == "23505": // unique_violation
				return dberror.ErrAlreadyExists.Msg("tangent already exists")
			}
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to insert tangent")
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}

func (mm *metadataManager) GetTangent(ctx context.Context, id uuid.UUID) (*models.Tangent, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)

	query := `
		SELECT id, info, public_key, status, tenant_id, created_at, updated_at
		FROM tangents
		WHERE id = $1
	`

	var tangent models.Tangent
	err := mm.conn().QueryRowContext(ctx, query, id).
		Scan(&tangent.ID, &tangent.Info, &tangent.PublicKey, &tangent.Status, &tangent.TenantID, &tangent.CreatedAt, &tangent.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("tangent not found")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}

	if tenantID != "" {
		if tangent.TenantID != string(tenantID) {
			log.Ctx(ctx).Error().Msgf("tangent %s is not in tenant %s", id, tenantID)
			return nil, dberror.ErrNotFound.Msg("tangent not found")
		}
	}

	return &tangent, nil
}

func (mm *metadataManager) UpdateTangent(ctx context.Context, tangent *models.Tangent) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	query := `
		UPDATE tangents
		SET info = $3,
			public_key = $4,
			status = $5,
			updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := mm.conn().ExecContext(ctx, query, tenantID, tangent.ID, tangent.Info, tangent.PublicKey, tangent.Status)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update tangent")
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("tangent not found")
	}

	return nil
}

func (mm *metadataManager) DeleteTangent(ctx context.Context, id uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	query := `
		DELETE FROM tangents
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := mm.conn().ExecContext(ctx, query, tenantID, id)
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("tangent not found")
	}

	return nil
}

func (mm *metadataManager) ListTangents(ctx context.Context) ([]*models.Tangent, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT id, info, public_key, status, tenant_id, created_at, updated_at
		FROM tangents
		WHERE tenant_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := mm.conn().QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}
	defer rows.Close()

	var result []*models.Tangent

	for rows.Next() {
		var tangent models.Tangent
		err := rows.Scan(&tangent.ID, &tangent.Info, &tangent.PublicKey, &tangent.Status, &tangent.TenantID, &tangent.CreatedAt, &tangent.UpdatedAt)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to scan tangent row")
			return nil, dberror.ErrDatabase.Err(err)
		}
		result = append(result, &tangent)
	}

	if err := rows.Err(); err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}

	return result, nil
}
