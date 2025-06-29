package db

import (
	"context"
	"testing"

	"github.com/jackc/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestCreateVariant(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant for testing
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	// Create the project for testing
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
	assert.NoError(t, err)
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	// Test case: Successfully create a variant
	variant := models.Variant{
		Name:        "test_variant",
		Description: "A test variant",
		CatalogID:   catalog.CatalogID,
		Info:        info,
	}
	err = DB(ctx).CreateVariant(ctx, &variant)
	assert.NoError(t, err)
	defer DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")

	// Verify that the variant was created successfully
	retrievedVariant, err := DB(ctx).GetVariant(ctx, catalog.CatalogID, variant.VariantID, "")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedVariant)
	assert.Equal(t, "test_variant", retrievedVariant.Name)

	// Test case: Create a variant with invalid name (should fail due to check constraint)
	invalidVariant := models.Variant{
		Name:        "invalid name with spaces",
		Description: "This variant should fail",
		Info:        info,
		CatalogID:   catalog.CatalogID,
	}
	err = DB(ctx).CreateVariant(ctx, &invalidVariant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)

	// Test case: Create a variant with invalid catalog ID (should fail)
	invalidCatalogID := models.Variant{
		Name:        "invalid_catalog_id",
		Description: "This variant should fail due to invalid catalog ID",
		Info:        info,
		CatalogID:   uuid.New(),
	}
	err = DB(ctx).CreateVariant(ctx, &invalidCatalogID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidCatalog)

	// Test case: Create a duplicate variant (should fail due to unique constraint)
	duplicateVariant := models.Variant{
		Name:        "test_variant",
		Description: "This variant should fail due to duplicate name",
		Info:        info,
		CatalogID:   catalog.CatalogID,
	}
	err = DB(ctx).CreateVariant(ctx, &duplicateVariant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrAlreadyExists)

	// Test case: Missing tenant ID in context (should fail)
	ctxWithoutTenant := catcommon.WithTenantID(ctx, "")
	err = DB(ctx).CreateVariant(ctxWithoutTenant, &variant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)
}

func TestGetVariant(t *testing.T) {
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
	assert.NoError(t, err)
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

	// Test case: Successfully retrieve a variant by ID
	retrievedVariant, err := DB(ctx).GetVariant(ctx, catalog.CatalogID, variant.VariantID, "")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedVariant)
	assert.Equal(t, "test_variant", retrievedVariant.Name)

	// Test case: Successfully retrieve a variant by name
	retrievedVariantByName, err := DB(ctx).GetVariant(ctx, catalog.CatalogID, uuid.Nil, "test_variant")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedVariantByName)
	assert.Equal(t, "test_variant", retrievedVariantByName.Name)

	// Test case: Retrieve a non-existent variant (should return ErrNotFound)
	nonExistentVariantID := uuid.New()
	_, err = DB(ctx).GetVariant(ctx, catalog.CatalogID, nonExistentVariantID, "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Test case: Missing tenant ID or project ID in context (should fail)
	ctxWithoutTenant := catcommon.WithTenantID(ctx, "")
	_, err = DB(ctx).GetVariant(ctxWithoutTenant, catalog.CatalogID, variant.VariantID, "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)

	// Test case: Neither variant ID nor name provided (should fail)
	_, err = DB(ctx).GetVariant(ctx, catalog.CatalogID, uuid.Nil, "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)
}

func TestUpdateVariant(t *testing.T) {
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

	// Create the catalog for testing
	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	assert.NoError(t, err)

	catalog := models.Catalog{
		Name:        "test_catalog",
		Description: "A test catalog",
		Info:        info,
	}
	err = DB(ctx).CreateCatalog(ctx, &catalog)
	assert.NoError(t, err)
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

	// Create another variant for testing name conflicts
	conflictingVariant := models.Variant{
		Name:        "conflicting_variant",
		Description: "A conflicting variant",
		CatalogID:   catalog.CatalogID,
		Info:        info,
	}
	err = DB(ctx).CreateVariant(ctx, &conflictingVariant)
	assert.NoError(t, err)

	// Test case: Successfully update a variant by ID
	updatedVariant := models.Variant{
		Name:        "updated_test_variant",
		Description: "An updated test variant",
		CatalogID:   catalog.CatalogID,
		Info:        info,
	}
	err = DB(ctx).UpdateVariant(ctx, variant.VariantID, "", &updatedVariant)
	assert.NoError(t, err)

	// Verify that the variant was updated successfully
	retrievedVariant, err := DB(ctx).GetVariant(ctx, catalog.CatalogID, variant.VariantID, "")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedVariant)
	assert.Equal(t, "updated_test_variant", retrievedVariant.Name)
	assert.Equal(t, "An updated test variant", retrievedVariant.Description)

	// Test case: Update a variant by name
	err = DB(ctx).UpdateVariant(ctx, uuid.Nil, "updated_test_variant", &updatedVariant)
	assert.NoError(t, err)

	// Test case: Update a non-existent variant (should return ErrNotFound)
	nonExistentVariantID := uuid.New()
	err = DB(ctx).UpdateVariant(ctx, nonExistentVariantID, "", &updatedVariant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Test case: Missing tenant ID in context (should fail)
	ctxWithoutTenant := catcommon.WithTenantID(ctx, "")
	err = DB(ctx).UpdateVariant(ctxWithoutTenant, variant.VariantID, "", &updatedVariant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)

	// Test case: Neither variant ID nor name provided (should fail)
	err = DB(ctx).UpdateVariant(ctx, uuid.Nil, "", &updatedVariant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)

	// Test case: Update variant with a conflicting name (should return ErrAlreadyExists)
	updatedVariantWithConflict := models.Variant{
		Name:        "conflicting_variant",
		Description: "An updated variant with a conflicting name",
		CatalogID:   catalog.CatalogID,
		Info:        info,
	}
	err = DB(ctx).UpdateVariant(ctx, variant.VariantID, "", &updatedVariantWithConflict)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrAlreadyExists)

	// Test case: Update variant with incorrect catalog ID (should return ErrInvalidCatalog)
	updatedVariantWithInvalidCatalog := models.Variant{
		Name:        "updated_test_variant",
		Description: "An updated variant with an invalid catalog ID",
		CatalogID:   uuid.New(),
		Info:        info,
	}
	err = DB(ctx).UpdateVariant(ctx, variant.VariantID, "", &updatedVariantWithInvalidCatalog)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestDeleteVariant(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	catalogID := uuid.New()

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

	// Create the catalog for testing
	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	assert.NoError(t, err)

	catalog := models.Catalog{
		CatalogID:   catalogID,
		Name:        "test_catalog",
		Description: "A test catalog",
		Info:        info,
	}
	err = DB(ctx).CreateCatalog(ctx, &catalog)
	assert.NoError(t, err)
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

	// Test case: Successfully delete a variant by ID
	err = DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")
	assert.NoError(t, err)

	// Verify that the variant was deleted successfully
	_, err = DB(ctx).GetVariant(ctx, catalog.CatalogID, variant.VariantID, "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Create another variant for testing deletion by name
	variant = models.Variant{
		Name:        "test_variant_to_delete",
		Description: "A variant to be deleted by name",
		CatalogID:   catalog.CatalogID,
		Info:        info,
	}
	err = DB(ctx).CreateVariant(ctx, &variant)
	assert.NoError(t, err)

	// Test case: Successfully delete a variant by name
	err = DB(ctx).DeleteVariant(ctx, catalog.CatalogID, uuid.Nil, "test_variant_to_delete")
	assert.NoError(t, err)

	// Verify that the variant was deleted successfully
	_, err = DB(ctx).GetVariant(ctx, catalog.CatalogID, uuid.Nil, "test_variant_to_delete")
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Test case: Delete a non-existent variant (should return no error as DELETE is idempotent)
	nonExistentVariantID := uuid.New()
	err = DB(ctx).DeleteVariant(ctx, catalog.CatalogID, nonExistentVariantID, "")
	assert.NoError(t, err)

	// Test case: Missing tenant ID in context (should fail)
	ctxWithoutTenant := catcommon.WithTenantID(ctx, "")
	err = DB(ctx).DeleteVariant(ctxWithoutTenant, catalog.CatalogID, variant.VariantID, "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)

	// Test case: Neither variant ID nor name provided (should fail)
	err = DB(ctx).DeleteVariant(ctx, catalog.CatalogID, uuid.Nil, "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)
}
