package postgresql

import (
	"context"
	"database/sql"
	"regexp"
	"strings"

	"encoding/json"

	"github.com/golang/snappy"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/config"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

func (om *objectManager) CreateSchemaDirectory(ctx context.Context, t catcommon.CatalogObjectType, dir *models.SchemaDirectory) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	if dir.DirectoryID == uuid.Nil {
		dir.DirectoryID = uuid.New()
	}
	if dir.VariantID == uuid.Nil {
		return dberror.ErrInvalidInput.Msg("variant_id cannot be empty")
	}
	if dir.TenantID == "" {
		return dberror.ErrInvalidInput.Msg("tenant_id cannot be empty")
	}
	if len(dir.Directory) == 0 {
		return dberror.ErrInvalidInput.Msg("directory cannot be nil")
	}

	dir.TenantID = tenantID

	tx, err := om.conn().BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to start transaction")
		return dberror.ErrDatabase.Err(err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	errDb := om.createSchemaDirectoryWithTransaction(ctx, t, dir, tx)
	if errDb != nil {
		tx.Rollback()
		return errDb
	}

	if err := tx.Commit(); err != nil {
		return dberror.ErrDatabase.Err(err)
	}
	return nil
}

func (om *objectManager) SetDirectory(ctx context.Context, t catcommon.CatalogObjectType, id uuid.UUID, dir []byte) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}

	query := `
		UPDATE ` + tableName + `
		SET directory = $1
		WHERE tenant_id = $2 AND directory_id = $3;`

	_, err := om.conn().ExecContext(ctx, query, dir, tenantID, id)
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}

func (om *objectManager) GetDirectory(ctx context.Context, t catcommon.CatalogObjectType, id uuid.UUID) ([]byte, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return nil, dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}

	query := `
		SELECT directory
		FROM ` + tableName + `
		WHERE tenant_id = $1 AND directory_id = $2;`

	var dir []byte
	err := om.conn().QueryRowContext(ctx, query, tenantID, id).Scan(&dir)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("directory not found")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}
	return dir, nil
}

func (om *objectManager) createSchemaDirectoryWithTransaction(ctx context.Context, t catcommon.CatalogObjectType, dir *models.SchemaDirectory, tx *sql.Tx) apperrors.Error {
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}
	if dir.DirectoryID == uuid.Nil {
		dir.DirectoryID = uuid.New()
	}

	// Insert the schema directory into the database and get created uuid
	query := ` INSERT INTO ` + tableName + ` (directory_id, variant_id, tenant_id, directory)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, directory_id) DO NOTHING RETURNING directory_id;`

	var directoryID uuid.UUID
	err := tx.QueryRowContext(ctx, query, dir.DirectoryID, dir.VariantID, dir.TenantID, dir.Directory).Scan(&directoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			return dberror.ErrAlreadyExists.Msg("schema directory already exists")
		} else {
			return dberror.ErrDatabase.Err(err)
		}
	}

	dir.DirectoryID = directoryID

	return nil
}

func (om *objectManager) GetSchemaDirectory(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID) (*models.SchemaDirectory, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return nil, dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}

	query := `SELECT directory_id, variant_id, tenant_id, directory
		FROM ` + tableName + `
		WHERE tenant_id = $1 AND directory_id = $2;`

	dir := &models.SchemaDirectory{}
	err := om.conn().QueryRowContext(ctx, query, tenantID, directoryID).Scan(&dir.DirectoryID, &dir.VariantID, &dir.TenantID, &dir.Directory)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("schema directory not found")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}
	return dir, nil
}

func (om *objectManager) GetObjectRefByPath(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string) (*models.ObjectRef, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return nil, dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}

	query := `
		SELECT directory-> $1 AS object
		FROM ` + tableName + `
		WHERE tenant_id = $2 AND directory_id = $3;`

	var objectData []byte
	err := om.conn().QueryRowContext(ctx, query, path, tenantID, directoryID).Scan(&objectData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("object not found in directory")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}

	if len(objectData) == 0 {
		return nil, dberror.ErrNotFound.Msg("object not found in directory")
	}

	var obj models.ObjectRef
	if err := json.Unmarshal(objectData, &obj); err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}

	return &obj, nil
}

func (om *objectManager) LoadObjectByPath(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string) (*models.CatalogObject, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return nil, dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}

	log.Ctx(ctx).Debug().Str("path", path).Str("DirectoryID", directoryID.String()).Msg("Loading object by path")
	query := `
		WITH hash_cte AS (
			SELECT (directory-> $1 ->> 'hash') AS hash
			FROM ` + tableName + `
			WHERE tenant_id = $2 AND directory_id = $3
		)
		SELECT
			co.hash,
			co.type,
			co.version,
			co.tenant_id,
			co.data
		FROM
			hash_cte
		JOIN
			catalog_objects co
		ON
			hash_cte.hash = co.hash
		WHERE
			co.tenant_id = $2;
	`

	var hash, version string
	var objType catcommon.CatalogObjectType
	var data []byte
	err := om.conn().QueryRowContext(ctx, query, path, tenantID, directoryID).Scan(&hash, &objType, &version, &tenantID, &data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("object not found in directory or catalog")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}

	// Create and populate the CatalogObject
	catalogObj := &models.CatalogObject{
		Hash:     hash,
		Type:     objType,
		Version:  version,
		TenantID: tenantID,
	}

	catalogObj.Data = data
	// Decompress the data
	if config.CompressCatalogObjects {
		catalogObj.Data, err = snappy.Decode(nil, data)
		if err != nil {
			return nil, dberror.ErrDatabase.Err(err)
		}
	}

	return catalogObj, nil
}

func (om *objectManager) AddOrUpdateObjectByPath(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string, obj models.ObjectRef) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}

	if !isValidPath(path) {
		return dberror.ErrInvalidInput.Msg("invalid path")
	}

	// Convert the object to JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	query := `
		UPDATE ` + tableName + `
		SET directory = jsonb_set(directory, ARRAY[$1], $2::jsonb)
		WHERE tenant_id = $3 AND directory_id = $4;`

	result, err := om.conn().ExecContext(ctx, query, path, data, tenantID, directoryID)
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	if rowsAffected == 0 {
		// No matching row was found with directory_id and tenant_id
		return dberror.ErrNotFound.Msg("object not found")
	}

	// get object to verify update
	if o, err := om.GetObjectRefByPath(ctx, t, directoryID, path); err != nil {
		return err
	} else if o.Hash != obj.Hash {
		return dberror.ErrDatabase.Msg("object hash mismatch after update")
	}

	return nil
}

func (om *objectManager) DeleteObjectByPath(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string) (catcommon.Hash, apperrors.Error) {
	var hash catcommon.Hash = ""
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return hash, dberror.ErrMissingTenantID
	}
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return hash, dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}
	log.Ctx(ctx).Debug().Str("path", path).Str("DirectoryID", directoryID.String()).Msg("Deleting object by path")
	// Update and return whether the key was removed
	query := `
		WITH to_delete AS (
			SELECT directory -> $1 ->> 'hash' AS deleted_hash
			FROM ` + tableName + `
			WHERE tenant_id = $2 AND directory_id = $3 AND directory ? $1
		)
		UPDATE ` + tableName + `
		SET directory = directory - $1
		WHERE tenant_id = $2 AND directory_id = $3 AND directory ? $1
		RETURNING (SELECT deleted_hash FROM to_delete);

	`
	var result sql.NullString
	err := om.conn().QueryRowContext(ctx, query, path, tenantID, directoryID).Scan(&result)
	if err == sql.ErrNoRows {
		return hash, nil // Key did not exist, so nothing was removed
	} else if err != nil {
		return hash, dberror.ErrDatabase.Err(err)
	} else if !result.Valid {
		return hash, dberror.ErrNotFound.Msg("object not found")
	}
	hash = catcommon.Hash(result.String)

	return hash, nil
}

func (om *objectManager) PathExists(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string) (bool, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return false, dberror.ErrMissingTenantID
	}
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return false, dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}

	query := `
		SELECT directory ? $1 AS exists
		FROM ` + tableName + `
		WHERE tenant_id = $2 AND directory_id = $3;`

	var exists bool
	err := om.conn().QueryRowContext(ctx, query, path, tenantID, directoryID).Scan(&exists)
	if err != nil {
		return false, dberror.ErrDatabase.Err(err)
	}

	return exists, nil
}

func getSchemaDirectoryTableName(t catcommon.CatalogObjectType) string {
	switch t {
	case catcommon.CatalogObjectTypeResource:
		return "resource_directory"
	case catcommon.CatalogObjectTypeSkillset:
		return "skillset_directory"
	default:
		return ""
	}
}

func isValidPath(path string) bool {
	var validPathPattern = regexp.MustCompile(`^(/[A-Za-z0-9_-]+)+$`)
	return validPathPattern.MatchString(path)
}

// DeleteNamespaceObjects deletes all objects in a namespace from the schema directory
func (om *objectManager) DeleteNamespaceObjects(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, namespace string) ([]string, apperrors.Error) {
	// Validate tenant and table
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}
	tableName := getSchemaDirectoryTableName(t)
	if tableName == "" {
		return nil, dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}

	tx, err := om.beginSerializableTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	dir, err := om.fetchDirectoryForUpdate(ctx, tx, tableName, tenantID, directoryID)
	if err != nil {
		return nil, err
	}

	directory, err := om.parseDirectory(dir)
	if err != nil {
		return nil, err
	}

	deletedPaths := om.deleteNamespacePaths(directory, namespace)
	if len(deletedPaths) == 0 {
		if err := om.commitTx(tx); err != nil {
			return nil, err
		}
		return nil, nil
	}

	updatedDir, err := om.marshalDirectory(directory)
	if err != nil {
		return nil, err
	}

	if err := om.updateDirectoryInTx(ctx, tx, tableName, updatedDir, directoryID, tenantID); err != nil {
		return nil, err
	}

	if err := om.commitTx(tx); err != nil {
		return nil, err
	}

	return deletedPaths, nil
}

func (om *objectManager) beginSerializableTx(ctx context.Context) (*sql.Tx, apperrors.Error) {
	tx, err := om.conn().BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to start transaction")
		return nil, dberror.ErrDatabase.Err(err)
	}
	return tx, nil
}

func (om *objectManager) fetchDirectoryForUpdate(ctx context.Context, tx *sql.Tx, tableName string, tenantID catcommon.TenantId, directoryID uuid.UUID) ([]byte, apperrors.Error) {
	var query string
	switch tableName {
	case "resource_directory":
		query = `SELECT directory FROM resource_directory WHERE tenant_id = $1 AND directory_id = $2 FOR UPDATE;`
	case "skillset_directory":
		query = `SELECT directory FROM skillset_directory WHERE tenant_id = $1 AND directory_id = $2 FOR UPDATE;`
	default:
		return nil, dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}
	var dir []byte
	err := tx.QueryRowContext(ctx, query, tenantID, directoryID).Scan(&dir)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("directory not found")
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to get directory")
		return nil, dberror.ErrDatabase.Err(err)
	}
	return dir, nil
}

func (om *objectManager) parseDirectory(dir []byte) (models.Directory, apperrors.Error) {
	directory, err := models.JSONToDirectory(dir)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal directory")
		return nil, dberror.ErrDatabase.Err(err)
	}
	return directory, nil
}

func (om *objectManager) deleteNamespacePaths(directory models.Directory, namespace string) []string {
	var deletedPaths []string
	namespacePrefix := "/--root--/" + namespace + "/"
	for path := range directory {
		if strings.HasPrefix(path, namespacePrefix) {
			delete(directory, path)
			deletedPaths = append(deletedPaths, path)
		}
	}
	return deletedPaths
}

func (om *objectManager) marshalDirectory(directory models.Directory) ([]byte, apperrors.Error) {
	updatedDir, err := models.DirectoryToJSON(directory)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal directory")
		return nil, dberror.ErrDatabase.Err(err)
	}
	return updatedDir, nil
}

func (om *objectManager) updateDirectoryInTx(ctx context.Context, tx *sql.Tx, tableName string, updatedDir []byte, directoryID uuid.UUID, tenantID catcommon.TenantId) apperrors.Error {
	var query string
	switch tableName {
	case "resource_directory":
		query = `UPDATE resource_directory SET directory = $1 WHERE directory_id = $2 AND tenant_id = $3;`
	case "skillset_directory":
		query = `UPDATE skillset_directory SET directory = $1 WHERE directory_id = $2 AND tenant_id = $3;`
	default:
		return dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}
	_, err := tx.ExecContext(ctx, query, updatedDir, directoryID, tenantID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update directory")
		return dberror.ErrDatabase.Err(err)
	}
	return nil
}

func (om *objectManager) commitTx(tx *sql.Tx) apperrors.Error {
	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("failed to commit transaction")
		return dberror.ErrDatabase.Err(err)
	}
	return nil
}
