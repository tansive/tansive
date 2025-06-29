package postgresql

import (
	"context"
	"database/sql"

	"github.com/golang/snappy"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/config"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
)

func (om *objectManager) CreateCatalogObject(ctx context.Context, obj *models.CatalogObject) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	if obj.Hash == "" {
		return dberror.ErrInvalidInput.Msg("hash cannot be empty")
	}
	if obj.Type == "" {
		return dberror.ErrInvalidInput.Msg("type cannot be empty")
	}
	if obj.Version == "" {
		return dberror.ErrInvalidInput.Msg("version cannot be empty")
	}
	if len(obj.Data) == 0 {
		return dberror.ErrInvalidInput.Msg("data cannot be nil")
	}

	// Ensure hash is at least 16 characters for hash_id
	if len(obj.Hash) < 16 {
		return dberror.ErrInvalidInput.Msg("hash must be at least 16 characters long")
	}

	obj.HashID = obj.Hash[:16]

	// snappy compress the data
	var dataZ []byte
	if config.CompressCatalogObjects {
		dataZ = snappy.Encode(nil, obj.Data)
		log.Ctx(ctx).Debug().Msgf("raw: %d, compressed: %d", len(obj.Data), len(dataZ))
	} else {
		dataZ = obj.Data // No compression
		log.Ctx(ctx).Debug().Msg("compression is disabled, using raw data")
	}

	// Insert the catalog object into the database
	query := `
		INSERT INTO catalog_objects (hash_id, hash, type, version, tenant_id, data)
		VALUES ($1, $2, $3, $4, $5, $6);
	`
	result, err := om.conn().ExecContext(ctx, query, obj.HashID, obj.Hash, obj.Type, obj.Version, tenantID, dataZ)
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	// Check if the row was inserted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	// If no rows were affected, it means the object already exists
	if rowsAffected == 0 {
		return dberror.ErrAlreadyExists.Msg("catalog object already exists")
	}

	return nil
}

func (om *objectManager) GetCatalogObject(ctx context.Context, hash string) (*models.CatalogObject, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}
	if hash == "" {
		return nil, dberror.ErrInvalidInput.Msg("hash cannot be empty")
	}

	// Ensure hash is at least 16 characters for hash_id
	if len(hash) < 16 {
		return nil, dberror.ErrInvalidInput.Msg("hash must be at least 16 characters long")
	}

	hashID := hash[:16]

	// Query to select catalog object based on composite key (hash_id, tenant_id) and exact hash match
	query := `
		SELECT hash_id, hash, type, version, tenant_id, data
		FROM catalog_objects
		WHERE tenant_id = $1 AND hash_id = $2 AND hash = $3
	`
	row := om.conn().QueryRowContext(ctx, query, tenantID, hashID, hash)

	var obj models.CatalogObject

	// Scan the result into obj fields - order must match SELECT columns
	err := row.Scan(&obj.HashID, &obj.Hash, &obj.Type, &obj.Version, &obj.TenantID, &obj.Data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dberror.ErrNotFound.Msg("catalog object not found")
		}
		return nil, dberror.ErrDatabase.Err(err)
	}

	// Uncompress the data
	if config.CompressCatalogObjects {
		obj.Data, err = snappy.Decode(nil, obj.Data)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to uncompress catalog object data")
			return nil, dberror.ErrDatabase.Err(err)
		}
	}

	return &obj, nil
}

func (om *objectManager) DeleteCatalogObject(ctx context.Context, t catcommon.CatalogObjectType, hash string) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	if hash == "" {
		return dberror.ErrInvalidInput.Msg("hash cannot be empty")
	}

	// Ensure hash is at least 16 characters for hash_id
	if len(hash) < 16 {
		return dberror.ErrInvalidInput.Msg("hash must be at least 16 characters long")
	}

	var table string
	switch t {
	case catcommon.CatalogObjectTypeResource:
		table = "resource_directory"
	case catcommon.CatalogObjectTypeSkillset:
		table = "skillset_directory"
	default:
		return dberror.ErrInvalidInput.Msg("invalid catalog object type")
	}

	// look for references in this table for this hash
	query := `
		SELECT 1
		FROM ` + table + `
		WHERE tenant_id = $1 AND jsonb_path_query_array(directory, '$.*.hash') @> to_jsonb($2::text)
		LIMIT 1;
	`
	var exists bool // we'll probably just hit the ErrNoRows case in case of false
	dberr := om.conn().QueryRowContext(ctx, query, tenantID, hash).Scan(&exists)
	if dberr != nil {
		if dberr != sql.ErrNoRows {
			return dberror.ErrDatabase.Err(dberr)
		}
	}
	if exists {
		// do nothing. There are other references to this object
		return nil
	}
	hashID := hash[:16]
	query = `
		DELETE FROM catalog_objects
		WHERE tenant_id = $1 AND hash_id = $2 AND hash = $3
	`
	result, err := om.conn().ExecContext(ctx, query, tenantID, hashID, hash)
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	// Check if the row was deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberror.ErrDatabase.Err(err)
	}

	// If no rows were affected, it means the object does not exist
	if rowsAffected == 0 {
		return dberror.ErrNotFound.Msg("catalog object not found")
	}

	return nil
}
