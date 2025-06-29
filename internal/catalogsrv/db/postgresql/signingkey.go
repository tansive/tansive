package postgresql

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

// CreateSigningKey creates a new signing key in the database.
func (mm *metadataManager) CreateSigningKey(ctx context.Context, key *models.SigningKey) apperrors.Error {
	// If key is active, we need to deactivate any existing active keys in a transaction
	tx, errdb := mm.conn().BeginTx(ctx, &sql.TxOptions{})
	if errdb != nil {
		log.Ctx(ctx).Error().Err(errdb).Msg("failed to start transaction")
		return dberror.ErrDatabase.Err(errdb)
	}

	var txErr error
	defer func() {
		if txErr != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Ctx(ctx).Error().Err(rollbackErr).Msg("failed to rollback transaction")
			}
		}
	}()

	if key.IsActive {
		// Deactivate any existing active keys
		_, txErr = tx.ExecContext(ctx, `
			UPDATE signing_keys
			SET is_active = false, updated_at = NOW()
			WHERE is_active = true`)
		if txErr != nil {
			log.Ctx(ctx).Error().Err(txErr).Msg("failed to deactivate existing keys")
			return dberror.ErrDatabase.Err(txErr)
		}
	}

	// Generate new UUID if not provided
	if key.KeyID == uuid.Nil {
		key.KeyID = uuid.New()
	}

	// Insert the new key
	query := `
		INSERT INTO signing_keys (key_id, public_key, private_key, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING key_id, created_at, updated_at`

	row := tx.QueryRowContext(ctx, query, key.KeyID, key.PublicKey, key.PrivateKey, key.IsActive)
	txErr = row.Scan(&key.KeyID, &key.CreatedAt, &key.UpdatedAt)
	if txErr != nil {
		log.Ctx(ctx).Error().Err(txErr).Msg("failed to create signing key")
		return dberror.ErrDatabase.Err(txErr)
	}

	if txErr = tx.Commit(); txErr != nil {
		log.Ctx(ctx).Error().Err(txErr).Msg("failed to commit transaction")
		return dberror.ErrDatabase.Err(txErr)
	}

	return nil
}

// GetSigningKey retrieves a signing key by its ID.
func (mm *metadataManager) GetSigningKey(ctx context.Context, keyID uuid.UUID) (*models.SigningKey, apperrors.Error) {
	query := `
		SELECT key_id, public_key, private_key, is_active, created_at, updated_at
		FROM signing_keys
		WHERE key_id = $1`

	var key models.SigningKey
	row := mm.conn().QueryRowContext(ctx, query, keyID)
	errdb := row.Scan(&key.KeyID, &key.PublicKey, &key.PrivateKey, &key.IsActive, &key.CreatedAt, &key.UpdatedAt)
	if errdb != nil {
		if errdb == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("signing key not found")
		}
		log.Ctx(ctx).Error().Err(errdb).Msg("failed to get signing key")
		return nil, dberror.ErrDatabase.Err(errdb)
	}

	return &key, nil
}

// GetActiveSigningKey retrieves the active signing key.
func (mm *metadataManager) GetActiveSigningKey(ctx context.Context) (*models.SigningKey, apperrors.Error) {
	query := `
		SELECT key_id, public_key, private_key, is_active, created_at, updated_at
		FROM signing_keys
		WHERE is_active = true`

	var key models.SigningKey
	row := mm.conn().QueryRowContext(ctx, query)
	errdb := row.Scan(&key.KeyID, &key.PublicKey, &key.PrivateKey, &key.IsActive, &key.CreatedAt, &key.UpdatedAt)
	if errdb != nil {
		if errdb == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("no active signing key found")
		}
		log.Ctx(ctx).Error().Err(errdb).Msg("failed to get active signing key")
		return nil, dberror.ErrDatabase.Err(errdb)
	}

	return &key, nil
}

// UpdateSigningKeyActive updates the active status of a signing key.
func (mm *metadataManager) UpdateSigningKeyActive(ctx context.Context, keyID uuid.UUID, isActive bool) apperrors.Error {
	// Start a transaction since we need to handle the one-active-key rule
	tx, errdb := mm.conn().BeginTx(ctx, &sql.TxOptions{})
	if errdb != nil {
		log.Ctx(ctx).Error().Err(errdb).Msg("failed to start transaction")
		return dberror.ErrDatabase.Err(errdb)
	}

	var txErr error
	defer func() {
		if txErr != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Ctx(ctx).Error().Err(rollbackErr).Msg("failed to rollback transaction")
			}
		}
	}()

	if isActive {
		// Deactivate any existing active keys
		_, txErr = tx.ExecContext(ctx, `
			UPDATE signing_keys
			SET is_active = false, updated_at = NOW()
			WHERE is_active = true`)
		if txErr != nil {
			log.Ctx(ctx).Error().Err(txErr).Msg("failed to deactivate existing keys")
			return dberror.ErrDatabase.Err(txErr)
		}
	}

	// Update the specified key
	query := `
		UPDATE signing_keys
		SET is_active = $1, updated_at = NOW()
		WHERE key_id = $2
		RETURNING key_id`

	row := tx.QueryRowContext(ctx, query, isActive, keyID)
	var returnedKeyID uuid.UUID
	txErr = row.Scan(&returnedKeyID)
	if txErr != nil {
		if txErr == sql.ErrNoRows {
			return dberror.ErrNotFound.Msg("signing key not found")
		}
		log.Ctx(ctx).Error().Err(txErr).Msg("failed to update signing key")
		return dberror.ErrDatabase.Err(txErr)
	}

	if txErr = tx.Commit(); txErr != nil {
		log.Ctx(ctx).Error().Err(txErr).Msg("failed to commit transaction")
		return dberror.ErrDatabase.Err(txErr)
	}

	return nil
}

// DeleteSigningKey deletes a signing key by its ID.
func (mm *metadataManager) DeleteSigningKey(ctx context.Context, keyID uuid.UUID) apperrors.Error {
	query := `
		DELETE FROM signing_keys
		WHERE key_id = $1
		RETURNING key_id`

	row := mm.conn().QueryRowContext(ctx, query, keyID)
	var returnedKeyID uuid.UUID
	errdb := row.Scan(&returnedKeyID)
	if errdb != nil {
		if errdb == sql.ErrNoRows {
			return dberror.ErrNotFound.Msg("signing key not found")
		}
		log.Ctx(ctx).Error().Err(errdb).Msg("failed to delete signing key")
		return dberror.ErrDatabase.Err(errdb)
	}

	return nil
}
