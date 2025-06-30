package postgresql

import (
	"context"
	"errors"

	"encoding/json"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

func (om *objectManager) UpsertResource(ctx context.Context, rg *models.Resource, directoryID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	rg.TenantID = tenantID

	if rg.ID == uuid.Nil {
		rg.ID = uuid.New()
	}

	if directoryID == uuid.Nil {
		return dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	err := om.AddOrUpdateObjectByPath(ctx,
		catcommon.CatalogObjectTypeResource,
		directoryID,
		rg.Path,
		models.ObjectRef{
			Hash: rg.Hash,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (om *objectManager) GetResource(ctx context.Context, path string, variantID uuid.UUID, directoryID uuid.UUID) (*models.Resource, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return nil, dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	objRef, err := om.GetObjectRefByPath(ctx, catcommon.CatalogObjectTypeResource, directoryID, path)
	if err != nil {
		return nil, err
	}

	return &models.Resource{
		Path:      path,
		Hash:      objRef.Hash,
		VariantID: variantID,
		TenantID:  tenantID,
	}, nil
}

func (om *objectManager) GetResourceObject(ctx context.Context, path string, directoryID uuid.UUID) (*models.CatalogObject, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return nil, dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	return om.LoadObjectByPath(ctx, catcommon.CatalogObjectTypeResource, directoryID, path)
}

func (om *objectManager) UpdateResource(ctx context.Context, rg *models.Resource, directoryID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	objRef, err := om.GetObjectRefByPath(ctx, catcommon.CatalogObjectTypeResource, directoryID, rg.Path)
	if err != nil {
		return err
	}
	objRef.Hash = rg.Hash
	err = om.AddOrUpdateObjectByPath(ctx,
		catcommon.CatalogObjectTypeResource,
		directoryID,
		rg.Path,
		*objRef,
	)
	if err != nil {
		return err
	}
	return nil
}

func (om *objectManager) DeleteResource(ctx context.Context, path string, directoryID uuid.UUID) (string, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return "", dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return "", dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	deletedHash, err := om.DeleteObjectByPath(ctx, catcommon.CatalogObjectTypeResource, directoryID, path)
	if err != nil {
		return "", err
	}

	return string(deletedHash), nil
}

func (om *objectManager) UpsertResourceObject(ctx context.Context, rg *models.Resource, obj *models.CatalogObject, directoryID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	rg.TenantID = tenantID

	if directoryID == uuid.Nil {
		return dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	// First create/update the catalog object
	err := om.CreateCatalogObject(ctx, obj)
	if err != nil {
		// If the object already exists, that's fine - we can continue
		if !errors.Is(err, dberror.ErrAlreadyExists) {
			return err
		}
	}

	// Then add/update the directory entry
	err = om.AddOrUpdateObjectByPath(ctx,
		catcommon.CatalogObjectTypeResource,
		directoryID,
		rg.Path,
		models.ObjectRef{
			Hash: rg.Hash,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (om *objectManager) ListResources(ctx context.Context, directoryID uuid.UUID) ([]models.Resource, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return nil, dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	// Get the directory
	dir, err := om.GetSchemaDirectory(ctx, catcommon.CatalogObjectTypeResource, directoryID)
	if err != nil {
		return nil, err
	}

	resources := []models.Resource{}
	var directory models.Directory

	if err := json.Unmarshal(dir.Directory, &directory); err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}

	for path, objRef := range directory {
		resource := models.Resource{
			Path: path,
			Hash: objRef.Hash,
		}
		resources = append(resources, resource)
	}

	return resources, nil
}
