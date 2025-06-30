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

func (mm *metadataManager) CreateView(ctx context.Context, view *models.View) (err apperrors.Error) {

	if view.CreatedBy == "" {
		return dberror.ErrMissingUserContext.Msg("missing user context")
	}
	view.UpdatedBy = view.CreatedBy

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

	err = mm.createViewWithTransaction(ctx, view, tx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create view")
		return err
	}

	if errStd := tx.Commit(); errStd != nil {
		log.Ctx(ctx).Error().Err(errStd).Msg("failed to commit transaction")
		return dberror.ErrDatabase.Err(errStd)
	}

	return nil
}

func (mm *metadataManager) createViewWithTransaction(ctx context.Context, view *models.View, tx *sql.Tx) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	view.TenantID = tenantID
	view.ViewID = uuid.New() // Override anything already set

	// Treat empty string as NULL
	description := sql.NullString{String: view.Description, Valid: view.Description != ""}
	label := sql.NullString{String: view.Label, Valid: view.Label != ""}

	query := `
		INSERT INTO views (label, description, info, rules, catalog_id, tenant_id, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING view_id
	`

	err := tx.QueryRowContext(ctx, query,
		label,
		description,
		view.Info,
		view.Rules,
		view.CatalogID,
		view.TenantID,
		view.CreatedBy,
		view.UpdatedBy,
	).Scan(&view.ViewID)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch {
			case pgErr.Code == "23505":
				return dberror.ErrAlreadyExists.Msg("view already exists")
			case pgErr.Code == "23514" && pgErr.ConstraintName == "views_label_check":
				log.Ctx(ctx).Error().Str("label", view.Label).Msg("invalid view label format")
				return dberror.ErrInvalidInput.Msg("invalid view label format")
			}
		}
		log.Ctx(ctx).Error().Err(err).Str("label", view.Label).Msg("failed to insert view")
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}

func (mm *metadataManager) GetView(ctx context.Context, viewID uuid.UUID) (*models.View, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT 
			v.view_id,
			v.label,
			v.description,
			v.info,
			v.rules,
			v.catalog_id,
			v.tenant_id,
			c.name AS catalog
		FROM 
			views v
		JOIN 
			catalogs c ON v.catalog_id = c.catalog_id AND v.tenant_id = c.tenant_id
		WHERE 
			v.tenant_id = $1 AND 
			v.view_id = $2;
	`

	var view models.View
	err := mm.conn().QueryRowContext(ctx, query, tenantID, viewID).
		Scan(&view.ViewID, &view.Label, &view.Description, &view.Info, &view.Rules, &view.CatalogID, &view.TenantID, &view.Catalog)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("view not found")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}

	return &view, nil
}

func (mm *metadataManager) GetViewByLabel(ctx context.Context, label string, catalogID uuid.UUID) (*models.View, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT 
			v.view_id,
			v.label,
			v.description,
			v.info,
			v.rules,
			v.catalog_id,
			v.tenant_id,
			c.name AS catalog
		FROM 
			views v
		JOIN 
			catalogs c ON v.catalog_id = c.catalog_id AND v.tenant_id = c.tenant_id
		WHERE 
			v.tenant_id = $1 AND 
			v.catalog_id = $2 AND 
			v.label = $3;
	`

	var view models.View
	err := mm.conn().QueryRowContext(ctx, query, tenantID, catalogID, label).
		Scan(&view.ViewID, &view.Label, &view.Description, &view.Info, &view.Rules, &view.CatalogID, &view.TenantID, &view.Catalog)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("view not found")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}

	return &view, nil
}

func (mm *metadataManager) UpdateView(ctx context.Context, view *models.View) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	if view.UpdatedBy == "" {
		return dberror.ErrMissingUserContext.Msg("missing user context")
	}

	var query string
	var args []interface{}

	if view.ViewID != uuid.Nil {
		// Update by ID
		query = `
			UPDATE views
			SET description = $3,
				info = $4,
				rules = $5,
				updated_by = $6,
				updated_at = NOW()
			WHERE tenant_id = $1 AND view_id = $2
		`
		args = []any{tenantID, view.ViewID, view.Description, view.Info, view.Rules, view.UpdatedBy}
	} else if view.Label != "" && view.CatalogID != uuid.Nil {
		// Update by label and catalog
		query = `
			UPDATE views
			SET description = $4,
				info = $5,
				rules = $6,
				updated_by = $7,
				updated_at = NOW()
			WHERE tenant_id = $1 AND catalog_id = $2 AND label = $3
		`
		args = []any{tenantID, view.CatalogID, view.Label, view.Description, view.Info, view.Rules, view.UpdatedBy}
	} else {
		return dberror.ErrNotFound.Msg("view not found: neither ID nor (label, catalog) provided")
	}

	result, err := mm.conn().ExecContext(ctx, query, args...)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update view")
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("view not found")
	}

	return nil
}

func (mm *metadataManager) DeleteView(ctx context.Context, viewID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	query := `
		DELETE FROM views
		WHERE tenant_id = $1 AND view_id = $2
	`

	result, err := mm.conn().ExecContext(ctx, query, tenantID, viewID)
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("view not found")
	}

	return nil
}

func (mm *metadataManager) DeleteViewByLabel(ctx context.Context, label string, catalogID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	if label == "" {
		return dberror.ErrInvalidInput.Msg("label cannot be empty")
	}

	if catalogID == uuid.Nil {
		return dberror.ErrInvalidInput.Msg("catalog ID cannot be empty")
	}

	query := `
		DELETE FROM views
		WHERE tenant_id = $1 AND catalog_id = $2 AND label = $3
	`

	result, err := mm.conn().ExecContext(ctx, query, tenantID, catalogID, label)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to delete view")
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to retrieve result information")
		return dberror.ErrDatabase.Err(err)
	}

	if rowsAffected == 0 {
		log.Ctx(ctx).Info().Str("label", label).Str("catalog_id", catalogID.String()).Str("tenant_id", string(tenantID)).Msg("view not found")
		return dberror.ErrNotFound.Msg("view not found")
	}

	return nil
}

func (mm *metadataManager) ListViewsByCatalog(ctx context.Context, catalogID uuid.UUID) ([]*models.View, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT view_id, label, description, info, rules, catalog_id, tenant_id
		FROM views
		WHERE tenant_id = $1 AND catalog_id = $2
		ORDER BY label ASC
	`

	rows, err := mm.conn().QueryContext(ctx, query, tenantID, catalogID)
	if err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}
	defer rows.Close()

	var result []*models.View

	for rows.Next() {
		var view models.View
		var description sql.NullString
		err := rows.Scan(&view.ViewID, &view.Label, &description, &view.Info, &view.Rules, &view.CatalogID, &view.TenantID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to scan view row")
			return nil, dberror.ErrDatabase.Err(err)
		}
		if description.Valid {
			view.Description = description.String
		}
		result = append(result, &view)
	}

	if err := rows.Err(); err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}

	return result, nil
}
