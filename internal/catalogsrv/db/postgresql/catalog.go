package postgresql

import (
	"context"
	"database/sql"

	"github.com/jackc/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

// CreateCatalog inserts a new catalog into the database.
// If the catalog name already exists for the project and tenant, it returns an error.
func (mm *metadataManager) CreateCatalog(ctx context.Context, catalog *models.Catalog) (err apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	userContext := catcommon.GetUserContext(ctx)
	if userContext == nil || userContext.UserID == "" {
		return dberror.ErrMissingUserContext.Msg("missing user context")
	}
	principal := "user/" + userContext.UserID

	projectID := catcommon.GetProjectID(ctx)
	if projectID == "" {
		return dberror.ErrInvalidInput.Msg("project ID is required")
	}

	catalog.ProjectID = projectID
	catalogID := catalog.CatalogID
	if catalogID == uuid.Nil {
		catalogID = uuid.New()
	}

	// create a transaction
	tx, errdb := mm.conn().BeginTx(ctx, &sql.TxOptions{})
	if errdb != nil {
		log.Ctx(ctx).Error().Err(errdb).Msg("failed to start transaction")
		return dberror.ErrDatabase.Err(errdb)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Ctx(ctx).Error().Err(rollbackErr).Msg("failed to rollback transaction")
			}
		}
	}()

	query := `
		INSERT INTO catalogs (catalog_id, name, description, info, tenant_id, project_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, project_id, name) DO NOTHING
		RETURNING catalog_id, name;
	`

	// Execute the query directly using mm.conn().QueryRowContext
	row := tx.QueryRowContext(ctx, query, catalogID, catalog.Name, catalog.Description, catalog.Info, tenantID, projectID)
	var insertedCatalogID uuid.UUID
	var insertedName string
	errDb := row.Scan(&insertedCatalogID, &insertedName)
	if errDb != nil {
		tx.Rollback()
		if errDb == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("name", catalog.Name).Str("catalog_id", catalog.CatalogID.String()).Msg("catalog already exists")
			return dberror.ErrAlreadyExists.Msg("catalog already exists")
		}
		log.Ctx(ctx).Error().Err(errDb).Str("name", catalog.Name).Str("catalog_id", catalog.CatalogID.String()).Msg("failed to insert catalog")
		return dberror.ErrDatabase.Err(errDb)
	}
	catalog.CatalogID = insertedCatalogID

	// create default variant
	variant := models.Variant{
		Name:        catcommon.DefaultVariant,
		CatalogID:   catalog.CatalogID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
		Description: "default variant",
	}
	err = mm.createVariantWithTransaction(ctx, &variant, tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	// create a default admin view
	viewDefJSON := `
	{
		"scope": {
			"catalog": "` + catalog.Name + `"
		},
		"rules": [
			{
				"intent": "Allow",
				"actions": ["system.catalog.admin"],
				"targets": []
			}
		]
	}
	`

	view := models.View{
		Label:       catcommon.DefaultAdminViewLabel,
		CatalogID:   catalog.CatalogID,
		Description: "default admin view",
		Info:        nil,
		Rules:       []byte(viewDefJSON),
		CreatedBy:   principal,
		UpdatedBy:   principal,
	}
	err = mm.createViewWithTransaction(ctx, &view, tx)

	// Commit the transaction if both insertions succeed
	if err := tx.Commit(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to commit transaction")
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}

// GetCatalogIDByName retrieves the catalog ID associated with a given catalog name and tenant ID.
func (mm *metadataManager) GetCatalogIDByName(ctx context.Context, catalogName string) (uuid.UUID, apperrors.Error) {
	var catalogID uuid.UUID

	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return uuid.Nil, dberror.ErrMissingTenantID
	}

	projectID := catcommon.GetProjectID(ctx)
	if projectID == "" {
		return uuid.Nil, dberror.ErrInvalidInput.Msg("project ID is required")
	}

	// Query to get the catalog_id by catalog name and tenant ID
	query := `
		SELECT catalog_id FROM catalogs 
		WHERE tenant_id = $1 AND project_id = $2 AND name = $3;
	`
	errDb := mm.conn().QueryRowContext(ctx, query, tenantID, projectID, catalogName).Scan(&catalogID)
	if errDb != nil {
		if errDb == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("catalog_name", catalogName).Msg("catalog not found")
			return uuid.Nil, dberror.ErrInvalidCatalog
		}
		log.Ctx(ctx).Error().Err(errDb).Msg("failed to retrieve catalog ID")
		return uuid.Nil, dberror.ErrDatabase.Err(errDb)
	}

	return catalogID, nil
}

// GetCatalogByID retrieves a catalog from the database by its ID.
func (mm *metadataManager) GetCatalogByID(ctx context.Context, catalogID uuid.UUID) (*models.Catalog, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
        SELECT catalog_id, name, description, info, project_id
        FROM catalogs
        WHERE tenant_id = $1 AND catalog_id = $2;
    `

	row := mm.conn().QueryRowContext(ctx, query, tenantID, catalogID)

	// Scan the result into the catalog model
	var catalog models.Catalog
	errDb := row.Scan(&catalog.CatalogID, &catalog.Name, &catalog.Description, &catalog.Info, &catalog.ProjectID)
	if errDb != nil {
		if errDb == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("catalog_id", catalogID.String()).Msg("catalog not found")
			return nil, dberror.ErrNotFound.Msg("catalog not found")
		}
		log.Ctx(ctx).Error().Err(errDb).Str("catalog_id", catalogID.String()).Msg("failed to retrieve catalog")
		return nil, dberror.ErrDatabase.Err(errDb)
	}

	return &catalog, nil
}

// GetCatalogByName retrieves a catalog from the database by its name.
func (mm *metadataManager) GetCatalogByName(ctx context.Context, name string) (*models.Catalog, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	projectID := catcommon.GetProjectID(ctx)
	if projectID == "" {
		return nil, dberror.ErrInvalidInput.Msg("project ID is required")
	}

	query := `
        SELECT catalog_id, name, description, info, project_id
        FROM catalogs
        WHERE tenant_id = $1 AND project_id = $2 AND name = $3;
    `

	row := mm.conn().QueryRowContext(ctx, query, tenantID, projectID, name)

	// Scan the result into the catalog model
	var catalog models.Catalog
	errDb := row.Scan(&catalog.CatalogID, &catalog.Name, &catalog.Description, &catalog.Info, &catalog.ProjectID)
	if errDb != nil {
		if errDb == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("name", name).Msg("catalog not found")
			return nil, dberror.ErrNotFound.Msg("catalog not found")
		}
		log.Ctx(ctx).Error().Err(errDb).Str("name", name).Msg("failed to retrieve catalog")
		return nil, dberror.ErrDatabase.Err(errDb)
	}

	return &catalog, nil
}

// UpdateCatalog updates an existing catalog in the database.
// If both catalogID and name are provided, catalogID takes precedence.
func (mm *metadataManager) UpdateCatalog(ctx context.Context, catalog *models.Catalog) apperrors.Error {
	// Retrieve tenant and project IDs from context
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	projectID := catcommon.GetProjectID(ctx)
	if projectID == "" {
		return dberror.ErrInvalidInput.Msg("project ID is required")
	}

	// Validate input to ensure either catalogID or name is provided
	if catalog == nil || (catalog.CatalogID == uuid.Nil && catalog.Name == "") {
		log.Ctx(ctx).Error().Msg("catalogID or name must be provided")
		return dberror.ErrInvalidInput.Msg("catalogID or name must be provided")
	}

	// Construct the update query based on the provided input
	query := `
		UPDATE catalogs
		SET description = $4, info = $5
		WHERE `

	var row *sql.Row
	if catalog.CatalogID != uuid.Nil {
		query += "tenant_id = $1 AND project_id = $2 AND catalog_id = $3 RETURNING catalog_id, name;"
		row = mm.conn().QueryRowContext(ctx, query, tenantID, projectID, catalog.CatalogID, catalog.Description, catalog.Info)
	} else {
		query += "tenant_id = $1 AND project_id = $2 AND name = $3 RETURNING catalog_id, name;"
		row = mm.conn().QueryRowContext(ctx, query, tenantID, projectID, catalog.Name, catalog.Description, catalog.Info)
	}

	// Scan the updated values
	var updatedCatalogID, updatedName string
	errDb := row.Scan(&updatedCatalogID, &updatedName)
	if errDb != nil {
		if errDb == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("name", catalog.Name).Str("catalog_id", catalog.CatalogID.String()).Msg("catalog not found for update")
			return dberror.ErrNotFound.Msg("catalog not found for update")
		}
		log.Ctx(ctx).Error().Err(errDb).Str("name", catalog.Name).Str("catalog_id", catalog.CatalogID.String()).Msg("failed to update catalog")
		return dberror.ErrDatabase.Err(errDb)
	}

	return nil
}

// DeleteCatalog deletes a catalog from the database.
// If both catalogID and name are provided, catalogID takes precedence.
func (mm *metadataManager) DeleteCatalog(ctx context.Context, catalogID uuid.UUID, name string) apperrors.Error {
	// Retrieve tenant and project IDs from context
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	projectID := catcommon.GetProjectID(ctx)
	if projectID == "" {
		return dberror.ErrInvalidInput.Msg("project ID is required")
	}

	// Validate input to ensure either catalogID or name is provided
	if catalogID == uuid.Nil && name == "" {
		log.Ctx(ctx).Error().Msg("catalogID or name must be provided")
		return dberror.ErrInvalidInput.Msg("catalogID or name must be provided")
	}

	query := `
		DELETE FROM catalogs
		WHERE `

	if catalogID != uuid.Nil {
		query += "tenant_id = $1 AND project_id = $2 AND catalog_id = $3;"
		_, err := mm.conn().ExecContext(ctx, query, tenantID, projectID, catalogID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("catalog_id", catalogID.String()).Msg("failed to delete catalog")
			return dberror.ErrDatabase.Err(err)
		}
	} else {
		query += "tenant_id = $1 AND project_id = $2 AND name = $3;"
		_, err := mm.conn().ExecContext(ctx, query, tenantID, projectID, name)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("name", name).Msg("failed to delete catalog")
			return dberror.ErrDatabase.Err(err)
		}
	}

	return nil
}

// ListCatalogs retrieves all catalogs for the current tenant and project.
func (mm *metadataManager) ListCatalogs(ctx context.Context) ([]*models.Catalog, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	projectID := catcommon.GetProjectID(ctx)
	if projectID == "" {
		return nil, dberror.ErrInvalidInput.Msg("project ID is required")
	}

	query := `
		SELECT catalog_id, name, description, info, project_id
		FROM catalogs
		WHERE tenant_id = $1 AND project_id = $2
		ORDER BY name ASC
	`

	rows, err := mm.conn().QueryContext(ctx, query, tenantID, projectID)
	if err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}
	defer rows.Close()

	var catalogs []*models.Catalog

	for rows.Next() {
		var catalog models.Catalog
		err := rows.Scan(&catalog.CatalogID, &catalog.Name, &catalog.Description, &catalog.Info, &catalog.ProjectID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to scan catalog row")
			return nil, dberror.ErrDatabase.Err(err)
		}
		catalogs = append(catalogs, &catalog)
	}

	if err := rows.Err(); err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}

	return catalogs, nil
}
