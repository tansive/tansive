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

func (om *objectManager) UpsertSkillSet(ctx context.Context, ss *models.SkillSet, directoryID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	ss.TenantID = tenantID

	if ss.ID == uuid.Nil {
		ss.ID = uuid.New()
	}

	if directoryID == uuid.Nil {
		return dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	err := om.AddOrUpdateObjectByPath(ctx,
		catcommon.CatalogObjectTypeSkillset,
		directoryID,
		ss.Path,
		models.ObjectRef{
			Hash:     ss.Hash,
			Metadata: ss.Metadata,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (om *objectManager) GetSkillSet(ctx context.Context, path string, variantID uuid.UUID, directoryID uuid.UUID) (*models.SkillSet, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return nil, dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	objRef, err := om.GetObjectRefByPath(ctx, catcommon.CatalogObjectTypeSkillset, directoryID, path)
	if err != nil {
		return nil, err
	}

	return &models.SkillSet{
		Path:      path,
		Hash:      objRef.Hash,
		Metadata:  objRef.Metadata,
		VariantID: variantID,
		TenantID:  tenantID,
	}, nil
}

func (om *objectManager) GetSkillSetObject(ctx context.Context, path string, directoryID uuid.UUID) (*models.CatalogObject, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return nil, dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	return om.LoadObjectByPath(ctx, catcommon.CatalogObjectTypeSkillset, directoryID, path)
}

func (om *objectManager) UpdateSkillSet(ctx context.Context, ss *models.SkillSet, directoryID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	objRef, err := om.GetObjectRefByPath(ctx, catcommon.CatalogObjectTypeSkillset, directoryID, ss.Path)
	if err != nil {
		return err
	}
	objRef.Hash = ss.Hash
	objRef.Metadata = ss.Metadata
	err = om.AddOrUpdateObjectByPath(ctx,
		catcommon.CatalogObjectTypeSkillset,
		directoryID,
		ss.Path,
		*objRef,
	)
	if err != nil {
		return err
	}
	return nil
}

func (om *objectManager) DeleteSkillSet(ctx context.Context, path string, directoryID uuid.UUID) (string, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return "", dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return "", dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	deletedHash, err := om.DeleteObjectByPath(ctx, catcommon.CatalogObjectTypeSkillset, directoryID, path)
	if err != nil {
		return "", err
	}

	return string(deletedHash), nil
}

func (om *objectManager) UpsertSkillSetObject(ctx context.Context, ss *models.SkillSet, obj *models.CatalogObject, directoryID uuid.UUID) apperrors.Error {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return dberror.ErrMissingTenantID
	}
	ss.TenantID = tenantID

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
		catcommon.CatalogObjectTypeSkillset,
		directoryID,
		ss.Path,
		models.ObjectRef{
			Hash:     ss.Hash,
			Metadata: ss.Metadata,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (om *objectManager) ListSkillSets(ctx context.Context, directoryID uuid.UUID) ([]models.SkillSet, apperrors.Error) {
	tenantID := catcommon.GetTenantID(ctx)
	if tenantID == "" {
		return nil, dberror.ErrMissingTenantID
	}

	if directoryID == uuid.Nil {
		return nil, dberror.ErrInvalidInput.Msg("invalid directory ID")
	}

	// Get the directory
	dir, err := om.GetSchemaDirectory(ctx, catcommon.CatalogObjectTypeSkillset, directoryID)
	if err != nil {
		return nil, err
	}

	skillsets := []models.SkillSet{}
	var directory models.Directory

	if err := json.Unmarshal(dir.Directory, &directory); err != nil {
		return nil, dberror.ErrDatabase.Err(err)
	}

	for path, objRef := range directory {
		skillset := models.SkillSet{
			Path:     path,
			Hash:     objRef.Hash,
			Metadata: objRef.Metadata,
		}
		skillsets = append(skillsets, skillset)
	}

	return skillsets, nil
}
