package catalogmanager

import (
	"errors"
	"path"
	"strings"
	"testing"

	"encoding/json"

	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	_ "github.com/tansive/tansive/internal/catalogsrv/schema/schemavalidator"
	"github.com/tansive/tansive/pkg/types"
)

func TestSkillSetValidation(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		expectedError bool
		errorTypes    []string // List of expected error messages
	}{
		{
			name: "valid skillset with multiple sources",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet",
				"metadata": {
					"name": "test-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default",
					"path": "/skillsets/test-skillset"
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
						},
						{
							"name": "python-runner",
							"runner": "system.pythonrunner",
							"config": {
								"module": "test_module",
								"function": "test_function"
							}
						}
					],
					"context": [
						{
							"name": "test-context",
							"provider": {
								"id": "system.redis",
								"config": {
									"host": "local.tansive.dev"
								}
							},
							"schema": {"type": "object"},
							"value": {"name": "John", "age": 30}
						}
					],
					"skills": [
						{
							"name": "test-skill-1",
							"description": "A test skill using command runner",
							"source": "command-runner",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
							"exportedActions": ["test.action1"]
						},
						{
							"name": "test-skill-2",
							"description": "A test skill using python runner",
							"source": "python-runner",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
							"exportedActions": ["test.action2"]
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
			expectedError: false,
		},
		{
			name: "invalid skillset - skill references non-existent runner",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet",
				"metadata": {
					"name": "test-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default",
					"path": "/skillsets/test-skillset"
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
					"skills": [
						{
							"name": "test-skill",
							"description": "A test skill",
							"source": "non-existent-runner",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
							"exportedActions": ["test.action"]
						}
					]
				}
			}`,
			expectedError: true,
			errorTypes:    []string{"skill test-skill has no runner"},
		},
		{
			name: "invalid skillset - missing runner source for skill",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet",
				"metadata": {
					"name": "test-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default",
					"path": "/skillsets/test-skillset"
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
					"skills": [
						{
							"name": "test-skill",
							"description": "A test skill",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
							"exportedActions": ["test.action"]
						}
					]
				}
			}`,
			expectedError: true,
			errorTypes:    []string{"SkillSet.Spec.Skills[0].Source: missing required attribute"},
		},
		{
			name: "invalid kind",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "InvalidKind",
				"metadata": {
					"name": "test-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default",
					"path": "/skillsets/test-skillset"
				}
			}`,
			expectedError: true,
			errorTypes:    []string{"unsupported kind"},
		},
		{
			name: "missing required fields",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet"
			}`,
			expectedError: true,
			errorTypes:    []string{"SkillSet.Metadata: missing required attribute"},
		},
		{
			name: "invalid skillset name format",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet",
				"metadata": {
					"name": "invalid name",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default",
					"path": "/skillsets/invalid-name"
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
							"provider": {
								"id": "system.redis",
								"config": {
									"host": "local.tansive.dev"
								}
							},
							"schema": {"type": "object"}
						}
					],
					"skills": [
						{
							"name": "test-skill",
							"description": "A test skill",
							"source": "command-runner",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
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
			expectedError: true,
			errorTypes:    []string{"invalid name"},
		},
		{
			name: "valid skillset with transform",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet",
				"metadata": {
					"name": "test-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default",
					"path": "/skillsets/test-skillset"
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
					"skills": [
						{
							"name": "test-skill",
							"description": "A test skill with transform",
							"source": "command-runner",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
							"transform": "function(input) { return input; }",
							"exportedActions": ["test.action"]
						}
					]
				}
			}`,
			expectedError: false,
		},
		{
			name: "invalid skillset - invalid transform syntax",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet",
				"metadata": {
					"name": "test-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default",
					"path": "/skillsets/test-skillset"
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
					"skills": [
						{
							"name": "test-skill",
							"description": "A test skill with invalid transform",
							"source": "command-runner",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
							"transform": "function(input) { invalid syntax }",
							"exportedActions": ["test.action"]
						}
					]
				}
			}`,
			expectedError: true,
			errorTypes:    []string{"skill test-skill transform"},
		},
		{
			name: "valid skillset with multiple skills and transforms",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet",
				"metadata": {
					"name": "test-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default",
					"path": "/skillsets/test-skillset"
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
					"skills": [
						{
							"name": "test-skill-1",
							"description": "A test skill without transform",
							"source": "command-runner",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
							"exportedActions": ["test.action1"]
						},
						{
							"name": "test-skill-2",
							"description": "A test skill with transform",
							"source": "command-runner",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
							"transform": "function(input) { return { ...input, processed: true }; }",
							"exportedActions": ["test.action2"]
						}
					]
				}
			}`,
			expectedError: false,
		},
		{
			name: "valid skillset with system.mcp.stdio runner",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "SkillSet",
				"metadata": {
					"name": "stdio-skillset",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default",
					"path": "/skillsets/stdio-skillset"
				},
				"spec": {
					"version": "1.0.0",
					"sources": [
						{
							"name": "stdio-runner",
							"runner": "system.mcp.stdio",
							"config": {}
						}
					],
					"skills": [
						{
							"name": "stdio-skill",
							"description": "A skill using system.mcp.stdio runner",
							"source": "stdio-runner",
							"inputSchema": {"type": "object"},
							"outputSchema": {"type": "object"},
							"exportedActions": ["stdio.action"]
						}
					]
				}
			}`,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s SkillSet
			err := json.Unmarshal([]byte(tt.jsonInput), &s)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			validationErrors := s.Validate()
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

func TestSkillSetManagerSave(t *testing.T) {
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

	// Create a valid skillset
	validJSON := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "SkillSet",
		"metadata": {
			"name": "test-skillset",
			"catalog": "test-catalog",
			"namespace": "default",
			"variant": "test-variant",
			"path": "/skillsets"
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
					"provider": {
						"id": "system.redis",
						"config": {
							"host": "local.tansive.dev"
						}
					}
				}
			],
			"skills": [
				{
					"name": "test-skill",
					"description": "A test skill",
					"source": "command-runner",
					"inputSchema": {"type": "object"},
					"outputSchema": {"type": "object"},
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
	}`

	manager := &skillSetManager{}
	err = json.Unmarshal([]byte(validJSON), &manager.skillSet)
	require.NoError(t, err)

	metadata := manager.skillSet.Metadata

	storagePath := metadata.GetStoragePath(catcommon.CatalogObjectTypeSkillset) + "/" + metadata.Name

	t.Run("Save - valid skillset", func(t *testing.T) {
		err := manager.Save(ctx)
		assert.NoError(t, err)

		// Verify the skillset was saved
		_, err = db.DB(ctx).GetSkillSet(ctx, storagePath, variant.VariantID, variant.SkillsetDirectoryID)
		assert.NoError(t, err)
	})

	t.Run("saves skillset with metadata", func(t *testing.T) {
		// ... existing test code until after save ...

		// Verify the skillset was saved with correct metadata
		ss, err := db.DB(ctx).GetSkillSet(ctx, storagePath, variant.VariantID, variant.SkillsetDirectoryID)
		require.NoError(t, err)
		require.NotNil(t, ss)
		require.NotEmpty(t, ss.Metadata)

		// Parse and verify metadata contents
		var metadata SkillMetadata
		require.NoError(t, json.Unmarshal(ss.Metadata, &metadata))

		// Verify skills in metadata
		require.Len(t, metadata.Skills, 1)
		require.Equal(t, "test-skill", metadata.Skills[0].Name)
		require.Equal(t, []policy.Action{"test.action"}, metadata.Skills[0].ExportedActions)

		// Verify dependencies in metadata
		require.Len(t, metadata.Dependencies, 1)
		require.Equal(t, "/resources/test", metadata.Dependencies[0].Path)
		require.Equal(t, KindResource, metadata.Dependencies[0].Kind)
		require.Equal(t, []policy.Action{"read"}, metadata.Dependencies[0].Actions)
	})

	t.Run("saves skillset with multiple skills in metadata", func(t *testing.T) {
		// Create a skillset with multiple skills
		ss := &SkillSet{
			ApiVersion: "0.1.0-alpha.1",
			Kind:       "SkillSet",
			Metadata: interfaces.Metadata{
				Name:      "multi-skill-set",
				Namespace: types.NullableStringFrom("default"),
				Path:      "/skillsets/multi-skill-set",
				Catalog:   "test-catalog",
				Variant:   types.NullableStringFrom("test-variant"),
			},
			Spec: SkillSetSpec{
				Version: "1.0.0",
				Sources: []SkillSetSource{
					{
						Name:   "command-runner",
						Runner: "system.commandrunner",
						Config: map[string]any{
							"command": "python3 skillsets/multi-skill-set.py",
						},
					},
				},
				Skills: []Skill{
					{
						Name:         "skill1",
						Description:  "First skill",
						Source:       "command-runner",
						InputSchema:  json.RawMessage(`{"type": "object"}`),
						OutputSchema: json.RawMessage(`{"type": "object"}`),
						ExportedActions: []policy.Action{
							"skill1.action1",
							"skill1.action2",
						},
					},
					{
						Name:         "skill2",
						Description:  "Second skill",
						Source:       "command-runner",
						InputSchema:  json.RawMessage(`{"type": "object"}`),
						OutputSchema: json.RawMessage(`{"type": "object"}`),
						ExportedActions: []policy.Action{
							"skill2.action1",
						},
					},
				},
				Dependencies: []Dependency{
					{
						Path:    "/resources/test-resource",
						Kind:    KindResource,
						Alias:   "test-resource",
						Actions: []policy.Action{"resource.action"},
					},
				},
			},
		}

		// Save the skillset
		sm := &skillSetManager{skillSet: *ss}
		err := sm.Save(ctx)
		require.NoError(t, err)

		// Verify the skillset was saved with correct metadata
		pathWithName := path.Clean(ss.Metadata.GetStoragePath(catcommon.CatalogObjectTypeSkillset) + "/" + ss.Metadata.Name)
		savedSS, err := db.DB(ctx).GetSkillSet(ctx, pathWithName, variant.VariantID, variant.SkillsetDirectoryID)
		require.NoError(t, err)
		require.NotNil(t, savedSS)
		require.NotEmpty(t, savedSS.Metadata)

		// Parse and verify metadata contents
		var metadata SkillMetadata
		require.NoError(t, json.Unmarshal(savedSS.Metadata, &metadata))

		// Verify skills in metadata
		require.Len(t, metadata.Skills, 2)

		// Verify first skill
		require.Equal(t, "skill1", metadata.Skills[0].Name)
		require.Equal(t, []policy.Action{"skill1.action1", "skill1.action2"}, metadata.Skills[0].ExportedActions)

		// Verify second skill
		require.Equal(t, "skill2", metadata.Skills[1].Name)
		require.Equal(t, []policy.Action{"skill2.action1"}, metadata.Skills[1].ExportedActions)

		// Verify dependencies
		require.Len(t, metadata.Dependencies, 1)
		require.Equal(t, "/resources/test-resource", metadata.Dependencies[0].Path)
		require.Equal(t, KindResource, metadata.Dependencies[0].Kind)
		require.Equal(t, []policy.Action{"resource.action"}, metadata.Dependencies[0].Actions)
	})
}

func TestSkillSetManagerDelete(t *testing.T) {
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

	metadata := &interfaces.Metadata{
		Name:      "test-skillset",
		Catalog:   "test-catalog",
		Namespace: types.NullableStringFrom("default"),
		Variant:   types.NullableStringFrom("test-variant"),
	}

	t.Run("Delete - non-existent skillset", func(t *testing.T) {
		err := DeleteSkillSet(ctx, metadata)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrObjectNotFound))
	})

	t.Run("Delete - valid skillset", func(t *testing.T) {
		// First create a skillset
		validJSON := `{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "SkillSet",
			"metadata": {
				"name": "test-skillset",
				"catalog": "test-catalog",
				"namespace": "default",
				"variant": "test-variant"
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
						"provider": {
							"id": "system.redis",
							"config": {
								"host": "local.tansive.dev"
							}
						}
					}
				],
				"skills": [
					{
						"name": "test-skill",
						"description": "A test skill",
						"source": "command-runner",
						"inputSchema": {"type": "object"},
						"outputSchema": {"type": "object"},
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
		}`

		manager := &skillSetManager{}
		err = json.Unmarshal([]byte(validJSON), &manager.skillSet)
		require.NoError(t, err)

		err = manager.Save(ctx)
		require.NoError(t, err)

		// Now delete it
		err = DeleteSkillSet(ctx, metadata)
		assert.NoError(t, err)

		// Verify it's gone
		path := getSkillSetStoragePath(metadata)
		_, err = db.DB(ctx).GetSkillSet(ctx, path, variant.VariantID, variant.SkillsetDirectoryID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, dberror.ErrNotFound))
	})
}

func TestSkillValidateInput(t *testing.T) {
	tests := []struct {
		name          string
		skill         Skill
		input         string
		expectedError bool
	}{
		{
			name: "valid input matches schema",
			skill: Skill{
				Name:         "test-skill",
				Description:  "A test skill",
				Source:       "command-runner",
				InputSchema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "number"}}, "required": ["name"]}`),
				OutputSchema: json.RawMessage(`{"type": "object"}`),
				ExportedActions: []policy.Action{
					"test.action",
				},
			},
			input:         `{"name": "John", "age": 30}`,
			expectedError: false,
		},
		{
			name: "invalid input - missing required field",
			skill: Skill{
				Name:         "test-skill",
				Description:  "A test skill",
				Source:       "command-runner",
				InputSchema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "number"}}, "required": ["name"]}`),
				OutputSchema: json.RawMessage(`{"type": "object"}`),
				ExportedActions: []policy.Action{
					"test.action",
				},
			},
			input:         `{"age": 30}`,
			expectedError: true,
		},
		{
			name: "invalid input - wrong type",
			skill: Skill{
				Name:         "test-skill",
				Description:  "A test skill",
				Source:       "command-runner",
				InputSchema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "number"}}, "required": ["name"]}`),
				OutputSchema: json.RawMessage(`{"type": "object"}`),
				ExportedActions: []policy.Action{
					"test.action",
				},
			},
			input:         `{"name": "John", "age": "thirty"}`,
			expectedError: true,
		},
		{
			name: "valid input with transform field",
			skill: Skill{
				Name:         "test-skill",
				Description:  "A test skill with transform",
				Source:       "command-runner",
				InputSchema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "number"}}, "required": ["name"]}`),
				OutputSchema: json.RawMessage(`{"type": "object"}`),
				Transform:    types.NullableStringFrom("function(input) { return input; }"),
				ExportedActions: []policy.Action{
					"test.action",
				},
			},
			input:         `{"name": "John", "age": 30}`,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make(map[string]any)
			err := json.Unmarshal([]byte(tt.input), &input)
			require.NoError(t, err)
			err = tt.skill.ValidateInput(input)
			if tt.expectedError {
				assert.Error(t, err, "Expected validation error but got none")
			} else {
				assert.NoError(t, err, "Unexpected validation error: %v", err)
			}
		})
	}
}

func TestSkillSetManagerContextOperations(t *testing.T) {
	validJSON := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "SkillSet",
		"metadata": {
			"name": "test-skillset",
			"catalog": "test-catalog",
			"namespace": "default",
			"variant": "default",
			"path": "/skillsets/test-skillset"
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
							"name": {"type": "string"},
							"age": {"type": "number"}
						},
						"required": ["name"]
					},
					"value": {"name": "John", "age": 30}
				}
			],
			"skills": [
				{
					"name": "test-skill",
					"description": "A test skill",
					"source": "command-runner",
					"inputSchema": {"type": "object"},
					"outputSchema": {"type": "object"},
					"exportedActions": ["test.action"]
				}
			]
		}
	}`

	manager := &skillSetManager{}
	err := json.Unmarshal([]byte(validJSON), &manager.skillSet)
	require.NoError(t, err)

	t.Run("GetContextValue - existing context", func(t *testing.T) {
		value, appErr := manager.GetContextValue("test-context")
		assert.NoError(t, appErr)
		assert.False(t, value.IsNil())

		var result map[string]any
		err := value.GetAs(&result)
		assert.NoError(t, err)
		assert.Equal(t, "John", result["name"])
		assert.Equal(t, float64(30), result["age"])
	})

	t.Run("GetContextValue - non-existent context", func(t *testing.T) {
		value, appErr := manager.GetContextValue("non-existent")
		assert.Error(t, appErr)
		assert.True(t, value.IsNil())
	})

	t.Run("SetContextValue - valid value", func(t *testing.T) {
		newValue, err := types.NullableAnyFrom(map[string]any{
			"name": "Jane",
			"age":  25,
		})
		require.NoError(t, err)

		appErr := manager.SetContextValue("test-context", newValue)
		assert.NoError(t, appErr)

		// Verify the value was set
		value, appErr := manager.GetContextValue("test-context")
		assert.NoError(t, appErr)
		assert.False(t, value.IsNil())

		var result map[string]any
		err = value.GetAs(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Jane", result["name"])
		assert.Equal(t, float64(25), result["age"])
	})

	t.Run("SetContextValue - invalid value (missing required field)", func(t *testing.T) {
		newValue, err := types.NullableAnyFrom(map[string]any{
			"age": 25, // Missing required "name" field
		})
		require.NoError(t, err)

		appErr := manager.SetContextValue("test-context", newValue)
		assert.Error(t, appErr)
	})

	t.Run("SetContextValue - invalid value (wrong type)", func(t *testing.T) {
		newValue, err := types.NullableAnyFrom(map[string]any{
			"name": "Jane",
			"age":  "not-a-number", // age should be a number
		})
		require.NoError(t, err)

		appErr := manager.SetContextValue("test-context", newValue)
		assert.Error(t, appErr)
	})

	t.Run("SetContextValue - non-existent context", func(t *testing.T) {
		newValue, err := types.NullableAnyFrom(map[string]any{
			"name": "Jane",
			"age":  25,
		})
		require.NoError(t, err)

		appErr := manager.SetContextValue("non-existent", newValue)
		assert.Error(t, appErr)
	})
}
