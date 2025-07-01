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

// CreateVariant creates a new variant in the database.
// It generates a new UUID for the variant ID and sets the project ID based on the context.
// If a variant with the same name and catalog ID already exists, the insertion is skipped.
// Returns an error if the variant already exists, the variant name format is invalid,
// the catalog ID is invalid, or there is a database error.
func (mm *metadataManager) CreateVariant(ctx context.Context, variant *models.Variant) (err apperrors.Error) {
	// Start a transaction
	tx, errdb := mm.conn().BeginTx(ctx, &sql.TxOptions{})
	if errdb != nil {
		log.Ctx(ctx).Error().Err(errdb).Msg("failed to start transaction")
		return dberror.ErrDatabase.Err(errdb)
	}
	defer func() {
		// Ensure transaction is rolled back if not committed
		if err != nil {
			tx.Rollback()
		}
	}()

	err = mm.createVariantWithTransaction(ctx, variant, tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction if both insertions succeed
	errdb = tx.Commit()
	if errdb != nil {
		log.Ctx(ctx).Error().Err(errdb).Msg("failed to commit transaction")
		return dberror.ErrDatabase.Err(errdb)
	}

	return nil
}

func (mm *metadataManager) createVariantWithTransaction(ctx context.Context, variant *models.Variant, tx *sql.Tx) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	// Prepare variant IDs and directory IDs
	variantID := variant.VariantID
	if variant.VariantID == uuid.Nil {
		variantID = uuid.New()
	}
	rgDirID := uuid.New()
	ssDirID := uuid.New()
	variant.ResourceDirectoryID = rgDirID
	variant.SkillsetDirectoryID = ssDirID

	// Insert variant
	if err := mm.insertVariantInTx(ctx, tx, variant, variantID, rgDirID, ssDirID, tenantID); err != nil {
		return err
	}

	// Create default namespace
	if err := mm.createDefaultNamespaceInTx(ctx, tx, variant, tenantID); err != nil {
		return err
	}

	// Create resource directory
	if err := mm.createResourceDirectoryInTx(ctx, tx, variant, rgDirID, tenantID); err != nil {
		return err
	}

	// Create skillset directory
	if err := mm.createSkillsetDirectoryInTx(ctx, tx, variant, ssDirID, tenantID); err != nil {
		return err
	}

	return nil
}

func (mm *metadataManager) insertVariantInTx(ctx context.Context, tx *sql.Tx, variant *models.Variant, variantID, rgDirID, ssDirID uuid.UUID, tenantID catcommon.TenantId) apperrors.Error {
	queryVariant := `
		INSERT INTO variants (variant_id, name, description, info, catalog_id, resource_directory, skillset_directory, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, catalog_id, name) DO NOTHING
		RETURNING variant_id, name;
	`

	row := tx.QueryRowContext(ctx, queryVariant, variantID, variant.Name, variant.Description, variant.Info, variant.CatalogID, rgDirID, ssDirID, tenantID)
	var insertedVariantID uuid.UUID
	var insertedName string
	err := row.Scan(&insertedVariantID, &insertedName)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("name", variant.Name).Str("variant_id", variant.VariantID.String()).Msg("variant already exists")
			return dberror.ErrAlreadyExists.Msg("variant already exists")
		}
		return mm.handleVariantInsertError(ctx, err, variant)
	}

	variant.VariantID = insertedVariantID
	return nil
}

func (mm *metadataManager) handleVariantInsertError(ctx context.Context, err error, variant *models.Variant) apperrors.Error {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		if pgErr.Code == "23514" && pgErr.ConstraintName == "variants_name_check" {
			log.Ctx(ctx).Error().Str("name", variant.Name).Msg("invalid variant name format")
			return dberror.ErrInvalidInput.Msg("invalid variant name format")
		}
		if pgErr.ConstraintName == "variants_catalog_id_fkey" {
			log.Ctx(ctx).Info().Str("catalog_id", variant.CatalogID.String()).Msg("catalog not found")
			return dberror.ErrInvalidCatalog
		}
		if pgErr.Code == "23503" || pgErr.ConstraintName == "variants_catalog_id_tenant_id_fkey" {
			log.Ctx(ctx).Error().Str("name", variant.Name).Msg("catalog not found or invalid")
			return dberror.ErrInvalidCatalog
		}
	}
	log.Ctx(ctx).Error().Err(err).Str("name", variant.Name).Str("variant_id", variant.VariantID.String()).Msg("failed to insert variant")
	return dberror.ErrDatabase.Err(err)
}

func (mm *metadataManager) createDefaultNamespaceInTx(ctx context.Context, tx *sql.Tx, variant *models.Variant, tenantID catcommon.TenantId) apperrors.Error {
	namespace := models.Namespace{
		Name:        catcommon.DefaultNamespace,
		VariantID:   variant.VariantID,
		TenantID:    tenantID,
		Description: "Default namespace for the variant",
		Info:        nil,
	}
	errDb := mm.createNamespaceWithTransaction(ctx, &namespace, tx)
	if errDb != nil {
		log.Ctx(ctx).Error().Err(errDb).Str("variant_id", variant.VariantID.String()).Msg("failed to create default namespace for variant")
		return errDb
	}
	return nil
}

func (mm *metadataManager) createResourceDirectoryInTx(ctx context.Context, tx *sql.Tx, variant *models.Variant, rgDirID uuid.UUID, tenantID catcommon.TenantId) apperrors.Error {
	dir := models.SchemaDirectory{
		DirectoryID: rgDirID,
		VariantID:   variant.VariantID,
		TenantID:    tenantID,
		Directory:   []byte("{}"),
	}

	tableName := getSchemaDirectoryTableName(catcommon.CatalogObjectTypeResource)
	if tableName == "" {
		return dberror.ErrInvalidInput.Msg("invalid catalog object type: resource group not supported")
	}

	return mm.insertDirectoryInTx(ctx, tx, dir, tableName, "resource groups directory")
}

func (mm *metadataManager) createSkillsetDirectoryInTx(ctx context.Context, tx *sql.Tx, variant *models.Variant, ssDirID uuid.UUID, tenantID catcommon.TenantId) apperrors.Error {
	ssDir := models.SchemaDirectory{
		DirectoryID: ssDirID,
		VariantID:   variant.VariantID,
		TenantID:    tenantID,
		Directory:   []byte("{}"),
	}

	tableName := getSchemaDirectoryTableName(catcommon.CatalogObjectTypeSkillset)
	if tableName == "" {
		return dberror.ErrInvalidInput.Msg("invalid catalog object type: skillset not supported")
	}

	return mm.insertDirectoryInTx(ctx, tx, ssDir, tableName, "skillset directory")
}

func (mm *metadataManager) insertDirectoryInTx(ctx context.Context, tx *sql.Tx, dir models.SchemaDirectory, tableName, directoryType string) apperrors.Error {
	query := ` INSERT INTO ` + tableName + ` (directory_id, variant_id, tenant_id, directory)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, directory_id) DO NOTHING RETURNING directory_id;`

	var directoryID uuid.UUID
	err := tx.QueryRowContext(ctx, query, dir.DirectoryID, dir.VariantID, dir.TenantID, dir.Directory).Scan(&directoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("directory_id", dir.DirectoryID.String()).Msgf("%s already exists, skipping", directoryType)
			return nil
		} else {
			return dberror.ErrDatabase.Err(err)
		}
	}
	dir.DirectoryID = directoryID
	return nil
}

// GetVariant retrieves a variant from the database based on the variant ID or name.
// If both variantID and name are provided, variantID takes precedence.
// Returns the variant if found, or an error if the variant is not found or there is a database error.
func (mm *metadataManager) GetVariant(ctx context.Context, catalogID uuid.UUID, variantID uuid.UUID, name string) (*models.Variant, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	var query string
	var row *sql.Row

	if variantID != uuid.Nil {
		query = `
			SELECT variant_id, name, description, info, catalog_id, resource_directory, skillset_directory
			FROM variants
			WHERE tenant_id = $1 AND variant_id = $2;
		`
		row = mm.conn().QueryRowContext(ctx, query, tenantID, variantID)
	} else if name != "" {
		query = `
			SELECT variant_id, name, description, info, catalog_id, resource_directory, skillset_directory
			FROM variants
			WHERE tenant_id = $1 AND catalog_id = $2 AND name = $3;
		`
		row = mm.conn().QueryRowContext(ctx, query, tenantID, catalogID, name)
	} else {
		log.Ctx(ctx).Error().Msg("either variant ID or name must be provided")
		return nil, dberror.ErrInvalidInput.Msg("either variant ID or name must be provided")
	}

	variant := &models.Variant{}
	err := row.Scan(&variant.VariantID, &variant.Name, &variant.Description, &variant.Info, &variant.CatalogID, &variant.ResourceDirectoryID, &variant.SkillsetDirectoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Msg("variant not found")
			return nil, dberror.ErrNotFound.Msg("variant not found")
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to retrieve variant")
		return nil, dberror.ErrDatabase.Err(err)
	}

	return variant, nil
}

// GetVariantByID retrieves a variant by its UUID. This function performs a direct lookup
func (mm *metadataManager) GetVariantByID(ctx context.Context, variantID uuid.UUID) (*models.Variant, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT variant_id, name, description, info, catalog_id, resource_directory, skillset_directory
		FROM variants
		WHERE tenant_id = $1 AND variant_id = $2;
	`
	row := mm.conn().QueryRowContext(ctx, query, tenantID, variantID)
	variant := &models.Variant{}
	err := row.Scan(&variant.VariantID, &variant.Name, &variant.Description, &variant.Info, &variant.CatalogID, &variant.ResourceDirectoryID, &variant.SkillsetDirectoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Msg("variant not found")
			return nil, dberror.ErrNotFound.Msg("variant not found")
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to retrieve variant")
		return nil, dberror.ErrDatabase.Err(err)
	}

	return variant, nil
}

func (mm *metadataManager) GetVariantIDFromName(ctx context.Context, catalogID uuid.UUID, name string) (uuid.UUID, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return uuid.Nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT variant_id
		FROM variants
		WHERE tenant_id = $1 AND catalog_id = $2 AND name = $3;
	`

	var variantID uuid.UUID
	err := mm.conn().QueryRowContext(ctx, query, tenantID, catalogID, name).Scan(&variantID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Msg("variant not found")
			return uuid.Nil, dberror.ErrNotFound.Msg("variant not found")
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to retrieve variant ID")
		return uuid.Nil, dberror.ErrDatabase.Err(err)
	}

	return variantID, nil
}

// UpdateVariant updates an existing variant in the database based on the variant ID or name.
// If both variantID and name are provided, variantID takes precedence.
// The VariantID and CatalogID fields cannot be updated.
// Returns an error if the variant is not found, the variant name already exists for the given catalog ID,
// the variant name format is invalid, or there is a database error.
func (mm *metadataManager) UpdateVariant(ctx context.Context, variantID uuid.UUID, name string, updatedVariant *models.Variant) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	var query string
	var row *sql.Row

	if variantID != uuid.Nil {
		query = `
			UPDATE variants
			SET name = $4, description = $5, info = $6
			WHERE tenant_id = $1 AND catalog_id = $2 AND variant_id = $3
			RETURNING variant_id;
		`
		row = mm.conn().QueryRowContext(ctx, query, tenantID, updatedVariant.CatalogID, variantID, updatedVariant.Name, updatedVariant.Description, updatedVariant.Info)
	} else if name != "" {
		query = `
			UPDATE variants
			SET name = $4, description = $5, info = $6
			WHERE tenant_id = $1 AND catalog_id = $2 AND name = $3
			RETURNING variant_id;
		`
		row = mm.conn().QueryRowContext(ctx, query, tenantID, updatedVariant.CatalogID, name, updatedVariant.Name, updatedVariant.Description, updatedVariant.Info)
	} else {
		log.Ctx(ctx).Error().Msg("either variant ID or name must be provided")
		return dberror.ErrInvalidInput.Msg("either variant ID or name must be provided")
	}

	var returnedVariantID uuid.UUID
	err := row.Scan(&returnedVariantID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Msg("variant not found or no changes made")
			return dberror.ErrNotFound.Msg("variant not found or no changes made")
		}
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "variants_tenant_id_catalog_id_name_key" { // Unique constraint violation
				log.Ctx(ctx).Error().Msg("variant name already exists for the given catalog_id")
				return dberror.ErrAlreadyExists.Msg("variant name already exists for the given catalog_id")
			}
			if pgErr.Code == "23514" && pgErr.ConstraintName == "variants_name_check" { // Check constraint violation code and specific constraint name
				log.Ctx(ctx).Error().Str("name", updatedVariant.Name).Msg("invalid variant name format")
				return dberror.ErrInvalidInput.Msg("invalid variant name format")
			}
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to update variant")
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}

// DeleteVariant deletes a variant from the database based on the variant ID or name.
// If both variantID and name are provided, variantID takes precedence.
// Returns an error if the variant is not found or there is a database error.
func (mm *metadataManager) DeleteVariant(ctx context.Context, catalogID, variantID uuid.UUID, name string) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	if variantID == uuid.Nil && (catalogID == uuid.Nil || name == "") {
		return dberror.ErrInvalidInput.Msg("either variant ID or name must be provided")
	}

	var query string
	var err error
	var result sql.Result

	if variantID != uuid.Nil {
		query = `
			DELETE FROM variants
			WHERE tenant_id = $1 AND catalog_id = $2 AND variant_id = $3;
		`
		result, err = mm.conn().ExecContext(ctx, query, tenantID, catalogID, variantID)
	} else {
		query = `
			DELETE FROM variants
			WHERE tenant_id = $1 AND catalog_id = $2 AND name = $3;
		`
		result, err = mm.conn().ExecContext(ctx, query, tenantID, catalogID, name)
	}

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to delete variant")
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to retrieve result information")
		return dberror.ErrDatabase.Err(err)
	}

	if rowsAffected == 0 {
		log.Ctx(ctx).Info().Str("variant_id", variantID.String()).Str("variant_name", name).Str("catalog_id", catalogID.String()).Str("tenant_id", string(tenantID)).Msg("variant not found")
	}

	return nil
}

func (mm *metadataManager) GetMetadataNames(ctx context.Context, catalogID uuid.UUID, variantID uuid.UUID) (string, string, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return "", "", dberror.ErrMissingTenantID
	}

	query := `
		SELECT catalog.name, variant.name
		FROM catalog
		JOIN variant ON catalog.catalog_id = variant.catalog_id AND catalog.tenant_id = variant.tenant_id
		WHERE catalog.tenant_id = $1 AND catalog.catalog_id = $2 AND variant.variant_id = $3;
	`

	var catalogName string
	var variantName string
	err := mm.conn().QueryRowContext(ctx, query, tenantID, catalogID, variantID).Scan(&catalogName, &variantName)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Msg("variant not found")
			return "", "", dberror.ErrNotFound.Msg("variant not found")
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to retrieve variant")
		return "", "", dberror.ErrDatabase.Err(err)
	}

	return catalogName, variantName, nil
}

// ListVariantsByCatalog retrieves all variants for a given catalog ID.
// Returns an array of VariantSummary containing just the variant ID, name, and directory IDs.
// Returns an error if there is a database error or if the tenant ID is missing.
func (mm *metadataManager) ListVariantsByCatalog(ctx context.Context, catalogID uuid.UUID) ([]models.VariantSummary, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	query := `
		SELECT variant_id, name, resource_directory, skillset_directory
		FROM variants
		WHERE tenant_id = $1 AND catalog_id = $2
		ORDER BY name;
	`

	rows, err := mm.conn().QueryContext(ctx, query, tenantID, catalogID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("catalog_id", catalogID.String()).Msg("failed to query variants")
		return nil, dberror.ErrDatabase.Err(err)
	}
	defer rows.Close()

	var variants []models.VariantSummary
	for rows.Next() {
		var variant models.VariantSummary
		err := rows.Scan(&variant.VariantID, &variant.Name, &variant.ResourceDirectoryID, &variant.SkillsetDirectoryID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to scan variant row")
			return nil, dberror.ErrDatabase.Err(err)
		}
		variants = append(variants, variant)
	}

	if err = rows.Err(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error iterating over variant rows")
		return nil, dberror.ErrDatabase.Err(err)
	}

	return variants, nil
}
