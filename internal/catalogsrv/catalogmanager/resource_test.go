package catalogmanager

import (
	"context"
	"path"
	"testing"

	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/pkg/types"
)

func TestResourceNewManager(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		metadata      *interfaces.Metadata
		expectedError bool
	}{
		{
			name: "valid resource",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "Resource",
				"metadata": {
					"name": "test-resource",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default"
				},
				"spec": {
					"schema": {"type": "integer"},
					"value": 42
				}
			}`,
			metadata: &interfaces.Metadata{
				Name:      "test-resource",
				Catalog:   "test-catalog",
				Namespace: types.NullableStringFrom("default"),
				Variant:   types.NullableStringFrom("default"),
			},
			expectedError: false,
		},
		{
			name:          "empty json",
			jsonInput:     "",
			metadata:      nil,
			expectedError: true,
		},
		{
			name: "invalid json",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "InvalidKind",
				"metadata": {
					"name": "test-resource",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default"
				}
			}`,
			metadata: &interfaces.Metadata{
				Name:      "test-resource",
				Catalog:   "test-catalog",
				Namespace: types.NullableStringFrom("default"),
				Variant:   types.NullableStringFrom("default"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewResourceManager(context.Background(), []byte(tt.jsonInput), tt.metadata)
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				if err != nil {
					t.Logf("Unexpected error: %v", err)
				}
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				if manager != nil {
					// Test Metadata() method
					metadata := manager.Metadata()
					assert.Equal(t, tt.metadata.Name, metadata.Name)
					assert.Equal(t, tt.metadata.Catalog, metadata.Catalog)
					assert.Equal(t, tt.metadata.Namespace, metadata.Namespace)
					assert.Equal(t, tt.metadata.Variant, metadata.Variant)

					// Test FullyQualifiedName() method
					expectedName := path.Clean(tt.metadata.Path + "/" + tt.metadata.Name)
					assert.Equal(t, expectedName, manager.FullyQualifiedName())
				}
			}
		})
	}
}

func TestLoadResourceManagerByPath(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := newDb()
	defer db.DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant and project for testing
	err := db.DB(ctx).CreateTenant(ctx, tenantID)
	require.NoError(t, err)
	defer db.DB(ctx).DeleteTenant(ctx, tenantID)

	err = db.DB(ctx).CreateProject(ctx, projectID)
	require.NoError(t, err)
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	require.NoError(t, err)

	// Create the catalog for testing
	catalog := models.Catalog{
		Name:        "test-catalog",
		Description: "A test catalog",
		Info:        info,
	}
	err = db.DB(ctx).CreateCatalog(ctx, &catalog)
	require.NoError(t, err)
	defer db.DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	// Set catalog ID in context
	ctx = catcommon.WithCatalogID(ctx, catalog.CatalogID)

	// Create a variant for testing
	variant := models.Variant{
		Name:        "test-variant",
		Description: "A test variant",
		CatalogID:   catalog.CatalogID,
		Info:        info,
	}
	err = db.DB(ctx).CreateVariant(ctx, &variant)
	require.NoError(t, err)
	defer db.DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")

	// Set variant ID and name in context
	ctx = catcommon.WithVariantID(ctx, variant.VariantID)
	ctx = catcommon.WithVariant(ctx, variant.Name)

	t.Run("Load existing resource", func(t *testing.T) {
		// Create a resource
		rsrcJson := []byte(`{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "test-resource",
				"description": "Test resource",
				"catalog": "test-catalog",
				"variant": "test-variant",
				"path": "/test"
			},
			"spec": {
				"schema": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						}
					}
				},
				"value": {
					"name": "test"
				}
			}
		}`)

		// Create and save the resource
		rm, err := NewResourceManager(ctx, rsrcJson, nil)
		require.NoError(t, err)
		err = rm.Save(ctx)
		require.NoError(t, err)

		// Create metadata for loading
		metadata := &interfaces.Metadata{
			Name:    "test-resource",
			Catalog: "test-catalog",
			Variant: types.NullableStringFrom("test-variant"),
			Path:    "/test",
		}

		// Load the resource by path
		loadedRm, err := LoadResourceManagerByPath(ctx, metadata)
		require.NoError(t, err)
		require.NotNil(t, loadedRm)

		// Verify metadata
		loadedMetadata := loadedRm.Metadata()
		assert.Equal(t, "test-resource", loadedMetadata.Name)
		assert.Equal(t, "test-catalog", loadedMetadata.Catalog)
		assert.Equal(t, "test-variant", loadedMetadata.Variant.String())
		assert.Equal(t, "/test", loadedMetadata.Path)

		// Verify value
		value := loadedRm.GetValue(ctx)
		assert.Equal(t, map[string]any{"name": "test"}, value.Get())

		// Verify storage path
		assert.Equal(t, rm.GetStoragePath(), loadedRm.GetStoragePath())
	})

	t.Run("Load non-existent resource", func(t *testing.T) {
		// Create metadata for non-existent resource
		metadata := &interfaces.Metadata{
			Name:    "non-existent",
			Catalog: "test-catalog",
			Variant: types.NullableStringFrom("test-variant"),
			Path:    "/test",
		}

		// Try to load a non-existent resource
		loadedRm, err := LoadResourceManagerByPath(ctx, metadata)
		assert.Error(t, err)
		assert.Nil(t, loadedRm)
	})

	t.Run("Load resource with invalid variant", func(t *testing.T) {
		// Create metadata for resource with invalid variant
		metadata := &interfaces.Metadata{
			Name:    "test-resource2",
			Catalog: "test-catalog",
			Variant: types.NullableStringFrom("invalid-variant"),
			Path:    "/test",
		}

		// Try to load a resource with invalid variant
		loadedRm, err := LoadResourceManagerByPath(ctx, metadata)
		assert.Error(t, err)
		assert.Nil(t, loadedRm)
	})
}
