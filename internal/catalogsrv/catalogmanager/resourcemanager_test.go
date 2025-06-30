package catalogmanager

import (
	"context"
	"strings"
	"testing"

	"encoding/json"

	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	_ "github.com/tansive/tansive/internal/catalogsrv/schema/schemavalidator"
	"github.com/tansive/tansive/pkg/types"
)

func TestResourceValidation(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		expectedError bool
		errorTypes    []string // List of expected error messages
	}{
		{
			name: "valid resource with schema and value",
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
					"schema": {"type": "boolean"},
					"value": true,
					"annotations": {
						"key1": "value1"
					}
				}
			}`,
			expectedError: false,
		},
		{
			name: "valid resource with schema, value and no policy",
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
					"schema": {"type": "boolean"},
					"value": true,
					"annotations": {
						"key1": "value1"
					}
				}
			}`,
			expectedError: false,
		},
		{
			name: "invalid kind",
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
			expectedError: true,
			errorTypes:    []string{"unsupported kind"},
		},
		{
			name: "missing required fields",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "Resource"
			}`,
			expectedError: true,
			errorTypes:    []string{"metadata: missing required attribute"},
		},
		{
			name: "invalid resource name format",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "Resource",
				"metadata": {
					"name": "invalid name",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default"
				},
				"spec": {
					"schema": {"type": "integer"},
					"value": 42
				}
			}`,
			expectedError: true,
			errorTypes:    []string{"invalid name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r Resource
			err := json.Unmarshal([]byte(tt.jsonInput), &r)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			validationErrors := r.Validate()
			if tt.expectedError {
				assert.NotEmpty(t, validationErrors, "Expected validation errors but got none")
				if len(tt.errorTypes) > 0 {
					errorMessages := make([]string, len(validationErrors))
					for i, err := range validationErrors {
						errorMessages[i] = err.Error()
					}
					for _, expectedErr := range tt.errorTypes {
						found := false
						for _, actualErr := range errorMessages {
							if strings.Contains(actualErr, expectedErr) {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected error message containing '%s' not found in validation errors: %v", expectedErr, errorMessages)
					}
				}
			} else {
				assert.Empty(t, validationErrors, "Unexpected validation errors: %v", validationErrors)
			}
		})
	}
}

func TestResourceManagerValueOperations(t *testing.T) {
	validJSON := `{
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
	}`

	metadata := &interfaces.Metadata{
		Name:      "test-resource",
		Catalog:   "test-catalog",
		Namespace: types.NullableStringFrom("default"),
		Variant:   types.NullableStringFrom("default"),
	}

	manager, err := NewResourceManager(context.Background(), []byte(validJSON), metadata)
	require.NoError(t, err)
	require.NotNil(t, manager)

	t.Run("GetValue - existing resource", func(t *testing.T) {
		value := manager.GetValue(context.Background())
		assert.Equal(t, float64(42), value.Get()) // JSON numbers are unmarshaled as float64
	})

	t.Run("GetValueJSON - existing resource", func(t *testing.T) {
		json, err := manager.GetValueJSON(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "42", string(json))
	})

	t.Run("SetValue - valid value", func(t *testing.T) {
		newValue, err := types.NullableAnyFrom(100)
		require.NoError(t, err)
		err = manager.SetValue(context.Background(), newValue)
		assert.NoError(t, err)

		value := manager.GetValue(context.Background())
		assert.Equal(t, float64(100), value.Get())
	})

	t.Run("SetValue - invalid value type", func(t *testing.T) {
		newValue, err := types.NullableAnyFrom("invalid")
		require.NoError(t, err)
		err = manager.SetValue(context.Background(), newValue)
		assert.Error(t, err)
	})
}

func TestResourceManagerSave(t *testing.T) {
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

	t.Run("Save basic resource", func(t *testing.T) {
		// Create a basic resource
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

		rm, err := NewResourceManager(ctx, rsrcJson, nil)
		require.NoError(t, err)

		// Save the resource
		err = rm.Save(ctx)
		require.NoError(t, err)

		// Verify the resource was saved
		rsrc, err := db.DB(ctx).GetResourceObject(ctx, rm.GetStoragePath(), variant.ResourceDirectoryID)
		require.NoError(t, err)
		assert.NotNil(t, rsrc)
		assert.Equal(t, rm.GetStoragePath(), rm.GetStoragePath())
	})

	t.Run("Save resource with different values", func(t *testing.T) {
		// Create first resource
		rsrcJson1 := []byte(`{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "test-resource2",
				"description": "Test resource 2",
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
					"name": "test1"
				}
			}
		}`)

		rm1, err := NewResourceManager(ctx, rsrcJson1, nil)
		require.NoError(t, err)

		// Save the first resource
		err = rm1.Save(ctx)
		require.NoError(t, err)

		// Get the first hash
		rsrc1, err := db.DB(ctx).GetResourceObject(ctx, rm1.GetStoragePath(), variant.ResourceDirectoryID)
		require.NoError(t, err)
		hash1 := rsrc1.Hash

		// Create second resource with different value
		rsrcJson2 := []byte(`{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "test-resource2",
				"description": "Test resource 2",
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
					"name": "test2"
				}
			}
		}`)

		rm2, err := NewResourceManager(ctx, rsrcJson2, nil)
		require.NoError(t, err)

		// Save the second resource
		err = rm2.Save(ctx)
		require.NoError(t, err)

		// Get the second hash
		rsrc2, err := db.DB(ctx).GetResourceObject(ctx, rm2.GetStoragePath(), variant.ResourceDirectoryID)
		require.NoError(t, err)
		hash2 := rsrc2.Hash

		// Verify hashes are different
		assert.NotEqual(t, hash1, hash2, "Hashes should be different for different values")
	})

	t.Run("Save resource with different schema", func(t *testing.T) {
		// Create first resource
		rsrcJson1 := []byte(`{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "test-resource3",
				"description": "Test resource 3",
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

		rm1, err := NewResourceManager(ctx, rsrcJson1, nil)
		require.NoError(t, err)

		// Save the first resource
		err = rm1.Save(ctx)
		require.NoError(t, err)

		// Get the first hash
		rsrc1, err := db.DB(ctx).GetResourceObject(ctx, rm1.GetStoragePath(), variant.ResourceDirectoryID)
		require.NoError(t, err)
		hash1 := rsrc1.Hash

		// Create second resource with different schema
		rsrcJson2 := []byte(`{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "test-resource3",
				"description": "Test resource 3",
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
						},
						"age": {
							"type": "number"
						}
					}
				},
				"value": {
					"name": "test",
					"age": 25
				}
			}
		}`)

		rm2, err := NewResourceManager(ctx, rsrcJson2, nil)
		require.NoError(t, err)

		// Save the second resource
		err = rm2.Save(ctx)
		require.NoError(t, err)

		// Get the second hash
		rsrc2, err := db.DB(ctx).GetResourceObject(ctx, rm2.GetStoragePath(), variant.ResourceDirectoryID)
		require.NoError(t, err)
		hash2 := rsrc2.Hash

		// Verify hashes are different
		assert.NotEqual(t, hash1, hash2, "Hashes should be different for different schemas")
	})
}

func TestResourceManagerDelete(t *testing.T) {
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

	t.Run("Delete existing resource", func(t *testing.T) {
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

		// Verify the resource exists
		rsrc, err := db.DB(ctx).GetResourceObject(ctx, rm.GetStoragePath(), variant.ResourceDirectoryID)
		require.NoError(t, err)
		assert.NotNil(t, rsrc)

		// Delete the resource
		metadata := rm.Metadata()
		err = DeleteResource(ctx, &metadata)
		require.NoError(t, err)

		// Verify the resource is deleted
		rsrc, err = db.DB(ctx).GetResourceObject(ctx, rm.GetStoragePath(), variant.ResourceDirectoryID)
		assert.Error(t, err)
		assert.Nil(t, rsrc)
	})

	t.Run("Delete non-existent resource", func(t *testing.T) {
		// Create a resource but don't save it
		rsrcJson := []byte(`{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "non-existent-resource",
				"description": "Non-existent resource",
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

		// Create the resource manager
		rm, err := NewResourceManager(ctx, rsrcJson, nil)
		require.NoError(t, err)

		// Try to delete the non-existent resource
		metadata := rm.Metadata()
		err = DeleteResource(ctx, &metadata)
		assert.Error(t, err)
	})

	t.Run("Delete resource with invalid variant", func(t *testing.T) {
		// Create a resource
		rsrcJson := []byte(`{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "test-resource2",
				"description": "Test resource 2",
				"catalog": "test-catalog",
				"variant": "invalid-variant",
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

		// Create the resource manager
		rm, err := NewResourceManager(ctx, rsrcJson, nil)
		require.NoError(t, err)

		// Try to delete with invalid variant
		metadata := rm.Metadata()
		err = DeleteResource(ctx, &metadata)
		assert.Error(t, err)
	})
}
