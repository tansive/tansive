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

func (mm *metadataManager) CreateNamespace(ctx context.Context, ns *models.Namespace) (err apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	ns.TenantID = tenantID

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

	err = mm.createNamespaceWithTransaction(ctx, ns, tx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create namespace")
		return err
	}

	if errStd := tx.Commit(); errStd != nil {
		log.Ctx(ctx).Error().Err(errStd).Msg("failed to commit transaction")
		return dberror.ErrDatabase.Err(errStd)
	}

	return nil
}

func (mm *metadataManager) createNamespaceWithTransaction(ctx context.Context, ns *models.Namespace, tx *sql.Tx) apperrors.Error {
	if ns.Name == "" {
		ns.Name = catcommon.DefaultNamespace
	}
	// Treat empty string as NULL
	description := sql.NullString{String: ns.Description, Valid: ns.Description != ""}

	query := `
		INSERT INTO namespaces (name, variant_id, tenant_id, description, info)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := tx.ExecContext(ctx, query,
		ns.Name,
		ns.VariantID,
		ns.TenantID,
		description,
		ns.Info,
	)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch {
			case pgErr.Code == "23505":
				return dberror.ErrAlreadyExists.Msg("namespace already exists")
			case pgErr.Code == "23514" && pgErr.ConstraintName == "namespaces_name_check":
				log.Ctx(ctx).Error().Str("name", ns.Name).Msg("invalid namespace name format")
				return dberror.ErrInvalidInput.Msg("invalid namespace name format")
			}
		}
		log.Ctx(ctx).Error().Err(err).Str("name", ns.Name).Msg("failed to insert namespace")
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}

func (mm *metadataManager) GetNamespace(ctx context.Context, name string, variantID uuid.UUID) (*models.Namespace, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}
	if name == "" {
		return nil, dberror.ErrInvalidInput.Msg("namespace name cannot be empty")
	}

	query := `
		SELECT 
			n.name, 
			n.variant_id, 
			n.tenant_id, 
			n.description, 
			n.info,
			c.catalog_id,
			c.name AS catalog,
			v.name AS variant
		FROM 
			namespaces n
		JOIN 
			variants v ON n.variant_id = v.variant_id AND n.tenant_id = v.tenant_id
		JOIN 
			catalogs c ON v.catalog_id = c.catalog_id AND v.tenant_id = c.tenant_id
		WHERE 
			n.tenant_id = $1 AND 
			n.variant_id = $2 AND 
			n.name = $3;
	`

	var ns models.Namespace
	err := mm.conn().QueryRowContext(ctx, query, tenantID, variantID, name).
		Scan(&ns.Name, &ns.VariantID, &ns.TenantID, &ns.Description, &ns.Info, &ns.CatalogID, &ns.Catalog, &ns.Variant)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("namespace not found")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}

	return &ns, nil
}

func (mm *metadataManager) UpdateNamespace(ctx context.Context, ns *models.Namespace) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	if ns.Name == "" {
		return dberror.ErrInvalidInput.Msg("namespace name cannot be empty")
	}

	query := `
		UPDATE namespaces
		SET description = $4,
		    info = $5,
		    updated_at = NOW()
		WHERE tenant_id = $1 AND variant_id = $2 AND name = $3
	`

	result, err := mm.conn().ExecContext(ctx, query, tenantID, ns.VariantID, ns.Name, ns.Description, ns.Info)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update namespace")
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("namespace not found")
	}

	return nil
}

func (mm *metadataManager) DeleteNamespace(ctx context.Context, name string, variantID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	if name == "" {
		return dberror.ErrInvalidInput.Msg("namespace name cannot be empty")
	}

	query := `
		DELETE FROM namespaces
		WHERE tenant_id = $1 AND variant_id = $2 AND name = $3
	`

	result, err := mm.conn().ExecContext(ctx, query, tenantID, variantID, name)
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("namespace not found")
	}

	return nil
}

func (mm *metadataManager) ListNamespacesByVariant(ctx context.Context, variantID uuid.UUID) ([]*models.Namespace, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT name, variant_id, tenant_id, description, info
		FROM namespaces
		WHERE tenant_id = $1 AND variant_id = $2
		ORDER BY name ASC
	`

	rows, err := mm.conn().QueryContext(ctx, query, tenantID, variantID)
	if err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}
	defer rows.Close()

	var result []*models.Namespace

	for rows.Next() {
		var ns models.Namespace
		err := rows.Scan(&ns.Name, &ns.VariantID, &ns.TenantID, &ns.Description, &ns.Info)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to scan namespace row")
			return nil, dberror.ErrDatabase.Err(err)
		}
		result = append(result, &ns)
	}

	if err := rows.Err(); err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}

	return result, nil
}
