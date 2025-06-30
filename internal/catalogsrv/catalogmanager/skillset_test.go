package catalogmanager

import (
	"context"
	"encoding/json"
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

func TestSkillSetNewManager(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		metadata      *interfaces.Metadata
		expectedError bool
	}{
		{
			name: "valid skillset",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet",
				"metadata": {
					"name": "test-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default"
				},
				"spec": {
					"version": "1.0.0",
					"sources": [
						{
							"name": "command-runner",
							"runner": "system.commandrunner",
							"config": {
								"command": "python3 test.py"
							}
						}
					],
					"context": [
						{
							"name": "test-context",
							"schema": {
								"type": "object",
								"properties": {
									"name": {
										"type": "string"
									}
								}
							}
						}
					],
					"skills": [
						{
							"name": "test-skill",
							"description": "Test skill",
							"source": "command-runner",
							"inputSchema": {
								"type": "object",
								"properties": {
									"input": {
										"type": "string"
									}
								}
							},
							"outputSchema": {
								"type": "object",
								"properties": {
									"output": {
										"type": "string"
									}
								}
							},
							"exportedActions": ["test.action"]
						}
					],
					"dependencies": [
						{
							"path": "/resources/test",
							"kind": "Resource",
							"alias": "test-resource",
							"actions": ["read"]
						}
					]
				}
			}`,
			metadata: &interfaces.Metadata{
				Name:      "test-skillset",
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
					"name": "test-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default"
				}
			}`,
			metadata: &interfaces.Metadata{
				Name:      "test-skillset",
				Catalog:   "test-catalog",
				Namespace: types.NullableStringFrom("default"),
				Variant:   types.NullableStringFrom("default"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewSkillSetManager(context.Background(), []byte(tt.jsonInput), tt.metadata)
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

func TestLoadSkillSetManagerByPath(t *testing.T) {
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

	t.Run("Load existing skillset", func(t *testing.T) {
		// Create a skillset
		skillsetJson := []byte(`{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "SkillSet",
			"metadata": {
				"name": "test-skillset",
				"description": "Test skillset",
				"catalog": "test-catalog",
				"variant": "test-variant",
				"path": "/test"
			},
			"spec": {
				"version": "1.0.0",
				"sources": [
					{
						"name": "command-runner",
						"runner": "system.commandrunner",
						"config": {
							"command": "python3 test.py"
						}
					}
				],
				"context": [
					{
						"name": "test-context",
						"schema": {
							"type": "object",
							"properties": {
								"name": {
									"type": "string"
								}
							}
						}
					}
				],
				"skills": [
					{
						"name": "test-skill",
						"description": "Test skill",
						"source": "command-runner",
						"inputSchema": {
							"type": "object",
							"properties": {
								"input": {
									"type": "string"
								}
							}
						},
						"outputSchema": {
							"type": "object",
							"properties": {
								"output": {
									"type": "string"
								}
							}
						},
						"exportedActions": ["test.action"]
					}
				],
				"dependencies": [
					{
						"path": "/resources/test",
						"kind": "Resource",
						"alias": "test-resource",
						"actions": ["read"]
					}
				]
			}
		}`)

		// Create and save the skillset
		sm, err := NewSkillSetManager(ctx, skillsetJson, nil)
		require.NoError(t, err)
		err = sm.Save(ctx)
		require.NoError(t, err)

		// Create metadata for loading
		metadata := &interfaces.Metadata{
			Name:    "test-skillset",
			Catalog: "test-catalog",
			Variant: types.NullableStringFrom("test-variant"),
			Path:    "/test",
		}

		// Load the skillset by path
		loadedSm, err := LoadSkillSetManagerByPath(ctx, metadata)
		require.NoError(t, err)
		require.NotNil(t, loadedSm)

		// Verify metadata
		loadedMetadata := loadedSm.Metadata()
		assert.Equal(t, "test-skillset", loadedMetadata.Name)
		assert.Equal(t, "test-catalog", loadedMetadata.Catalog)
		assert.Equal(t, "test-variant", loadedMetadata.Variant.String())
		assert.Equal(t, "/test", loadedMetadata.Path)

		// Verify storage path
		assert.Equal(t, sm.GetStoragePath(), loadedSm.GetStoragePath())
	})

	t.Run("Load non-existent skillset", func(t *testing.T) {
		// Create metadata for non-existent skillset
		metadata := &interfaces.Metadata{
			Name:    "non-existent",
			Catalog: "test-catalog",
			Variant: types.NullableStringFrom("test-variant"),
			Path:    "/test",
		}

		// Try to load a non-existent skillset
		loadedSm, err := LoadSkillSetManagerByPath(ctx, metadata)
		assert.Error(t, err)
		assert.Nil(t, loadedSm)
	})

	t.Run("Load skillset with invalid variant", func(t *testing.T) {
		// Create metadata for skillset with invalid variant
		metadata := &interfaces.Metadata{
			Name:    "test-skillset2",
			Catalog: "test-catalog",
			Variant: types.NullableStringFrom("invalid-variant"),
			Path:    "/test",
		}

		// Try to load a skillset with invalid variant
		loadedSm, err := LoadSkillSetManagerByPath(ctx, metadata)
		assert.Error(t, err)
		assert.Nil(t, loadedSm)
	})
}

func TestHashHiddenContextValues(t *testing.T) {
	// Create a test skillset with hidden context values
	skillsetJSON := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "SkillSet",
		"metadata": {
			"name": "test-skillset",
			"catalog": "test-catalog",
			"variant": "test-variant",
			"path": "/test"
		},
		"spec": {
			"version": "1.0.0",
			"sources": [
				{
					"name": "test-runner",
					"runner": "system.testrunner",
					"config": {}
				}
			],
			"context": [
				{
					"name": "visible-context",
					"schema": {"type": "string"},
					"value": "visible-value",
					"attributes": {
						"hidden": false
					}
				},
				{
					"name": "hidden-context",
					"schema": {"type": "string"},
					"value": "secret-password",
					"attributes": {
						"hidden": true
					}
				},
				{
					"name": "hidden-complex-context",
					"schema": {"type": "object"},
					"value": {"key": "secret-value", "nested": {"data": "sensitive"}},
					"attributes": {
						"hidden": true
					}
				},
				{
					"name": "null-context",
					"schema": {"type": "string"},
					"value": null,
					"attributes": {
						"hidden": true
					}
				}
			],
			"skills": [
				{
					"name": "test-skill",
					"description": "Test skill",
					"source": "test-runner",
					"inputSchema": {"type": "object"},
					"outputSchema": {"type": "object"},
					"exportedActions": ["test.action"]
				}
			]
		}
	}`

	// Create a skillset kind handler
	req := interfaces.RequestContext{
		Catalog:    "test-catalog",
		Variant:    "test-variant",
		Namespace:  "",
		ObjectPath: "/test",
		ObjectName: "test-skillset",
		ObjectType: catcommon.CatalogObjectTypeSkillset,
	}

	handler := &skillsetKindHandler{req: req}

	// Test the hashHiddenContextValues function
	jsonData := []byte(skillsetJSON)
	result, err := handler.hashHiddenContextValues(jsonData)
	require.NoError(t, err)

	// Parse the result to verify the changes
	var resultMap map[string]interface{}
	unmarshalErr := json.Unmarshal(result, &resultMap)
	require.NoError(t, unmarshalErr)

	// Get the context array
	spec := resultMap["spec"].(map[string]interface{})
	contexts := spec["context"].([]interface{})

	// Verify visible context is unchanged
	visibleContext := contexts[0].(map[string]interface{})
	assert.Equal(t, "visible-value", visibleContext["value"])

	// Verify hidden context is hashed
	hiddenContext := contexts[1].(map[string]interface{})
	hashedValue := hiddenContext["value"].(string)
	assert.NotEqual(t, "secret-password", hashedValue)
	assert.Len(t, hashedValue, 8) // Should be 8 characters

	// Verify hidden complex context is hashed
	hiddenComplexContext := contexts[2].(map[string]interface{})
	hashedComplexValue := hiddenComplexContext["value"].(string)
	assert.NotEqual(t, map[string]interface{}{"key": "secret-value", "nested": map[string]interface{}{"data": "sensitive"}}, hashedComplexValue)
	assert.Len(t, hashedComplexValue, 8) // Should be 8 characters

	// Verify null context is unchanged (null values should not be hashed)
	nullContext := contexts[3].(map[string]interface{})
	assert.Nil(t, nullContext["value"])
}

func TestHashHiddenContextValuesNoHiddenContexts(t *testing.T) {
	// Create a test skillset with no hidden context values
	skillsetJSON := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "SkillSet",
		"metadata": {
			"name": "test-skillset",
			"catalog": "test-catalog",
			"variant": "test-variant",
			"path": "/test"
		},
		"spec": {
			"version": "1.0.0",
			"sources": [
				{
					"name": "test-runner",
					"runner": "system.testrunner",
					"config": {}
				}
			],
			"context": [
				{
					"name": "visible-context",
					"schema": {"type": "string"},
					"value": "visible-value",
					"attributes": {
						"hidden": false
					}
				}
			],
			"skills": [
				{
					"name": "test-skill",
					"description": "Test skill",
					"source": "test-runner",
					"inputSchema": {"type": "object"},
					"outputSchema": {"type": "object"},
					"exportedActions": ["test.action"]
				}
			]
		}
	}`

	req := interfaces.RequestContext{
		Catalog:    "test-catalog",
		Variant:    "test-variant",
		Namespace:  "",
		ObjectPath: "/test",
		ObjectName: "test-skillset",
		ObjectType: catcommon.CatalogObjectTypeSkillset,
	}

	handler := &skillsetKindHandler{req: req}

	// Test the hashHiddenContextValues function
	jsonData := []byte(skillsetJSON)
	result, err := handler.hashHiddenContextValues(jsonData)
	require.NoError(t, err)

	// The result should be identical to the input since no contexts are hidden
	assert.Equal(t, skillsetJSON, string(result))
}

func TestHashHiddenContextValuesNoContexts(t *testing.T) {
	// Create a test skillset with no context values
	skillsetJSON := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "SkillSet",
		"metadata": {
			"name": "test-skillset",
			"catalog": "test-catalog",
			"variant": "test-variant",
			"path": "/test"
		},
		"spec": {
			"version": "1.0.0",
			"sources": [
				{
					"name": "test-runner",
					"runner": "system.testrunner",
					"config": {}
				}
			],
			"skills": [
				{
					"name": "test-skill",
					"description": "Test skill",
					"source": "test-runner",
					"inputSchema": {"type": "object"},
					"outputSchema": {"type": "object"},
					"exportedActions": ["test.action"]
				}
			]
		}
	}`

	req := interfaces.RequestContext{
		Catalog:    "test-catalog",
		Variant:    "test-variant",
		Namespace:  "",
		ObjectPath: "/test",
		ObjectName: "test-skillset",
		ObjectType: catcommon.CatalogObjectTypeSkillset,
	}

	handler := &skillsetKindHandler{req: req}

	// Test the hashHiddenContextValues function
	jsonData := []byte(skillsetJSON)
	result, err := handler.hashHiddenContextValues(jsonData)
	require.NoError(t, err)

	// The result should be identical to the input since there are no contexts
	assert.Equal(t, skillsetJSON, string(result))
}
