package db

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestResourceOperations(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant and project for testing
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	assert.NoError(t, err)

	// Create the catalog for testing
	catalog := models.Catalog{
		Name:        "test_catalog",
		Description: "A test catalog",
		Info:        info,
	}
	err = DB(ctx).CreateCatalog(ctx, &catalog)
	require.NoError(t, err)
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	// Create a variant for testing
	variant := models.Variant{
		Name:        "test_variant",
		Description: "A test variant",
		CatalogID:   catalog.CatalogID,
		Info:        info,
	}
	err = DB(ctx).CreateVariant(ctx, &variant)
	assert.NoError(t, err)
	defer DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")

	// Create a mock resource
	rg := &models.Resource{
		Path:      "/test/resource",
		Hash:      "test_hash_123456789012345",
		VariantID: variant.VariantID,
	}

	// Create a mock catalog object
	obj := &models.CatalogObject{
		Hash:     rg.Hash,
		Type:     catcommon.CatalogObjectTypeResource,
		Version:  "0.1.0-alpha.1",
		TenantID: tenantID,
		Data:     []byte(`{"key": "value"}`),
	}

	// Test UpsertResourceObject
	err = DB(ctx).UpsertResourceObject(ctx, rg, obj, variant.ResourceDirectoryID)
	require.NoError(t, err)

	// Test GetResource
	retrievedRG, err := DB(ctx).GetResource(ctx, rg.Path, variant.VariantID, variant.ResourceDirectoryID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedRG)
	assert.Equal(t, rg.Path, retrievedRG.Path)
	assert.Equal(t, rg.Hash, strings.TrimSpace(retrievedRG.Hash))
	assert.Equal(t, rg.VariantID, retrievedRG.VariantID)

	// Test GetResourceObject
	retrievedObj, err := DB(ctx).GetResourceObject(ctx, rg.Path, variant.ResourceDirectoryID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedObj)
	assert.Equal(t, obj.Hash, strings.TrimSpace(retrievedObj.Hash))
	assert.Equal(t, obj.Type, retrievedObj.Type)
	assert.Equal(t, obj.Version, retrievedObj.Version)

	// Test UpdateResource
	rg.Hash = "updated_hash_456789012345678"
	err = DB(ctx).UpdateResource(ctx, rg, variant.ResourceDirectoryID)
	assert.NoError(t, err)

	// Verify update
	updatedRG, err := DB(ctx).GetResource(ctx, rg.Path, variant.VariantID, variant.ResourceDirectoryID)
	assert.NoError(t, err)
	assert.NotNil(t, updatedRG)
	assert.Equal(t, rg.Hash, strings.TrimSpace(updatedRG.Hash))

	// Test DeleteResource
	deletedHash, err := DB(ctx).DeleteResource(ctx, rg.Path, variant.ResourceDirectoryID)
	assert.NoError(t, err)
	assert.Equal(t, rg.Hash, deletedHash)

	// Verify deletion
	_, err = DB(ctx).GetResource(ctx, rg.Path, variant.VariantID, variant.ResourceDirectoryID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Test error cases
	// Test with invalid directory ID
	_, err = DB(ctx).GetResource(ctx, rg.Path, variant.VariantID, uuid.Nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)

	// Test with missing tenant ID
	ctxWithoutTenant := catcommon.WithTenantID(ctx, "")
	_, err = DB(ctx).GetResource(ctxWithoutTenant, rg.Path, variant.VariantID, variant.ResourceDirectoryID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrMissingTenantID)
}
