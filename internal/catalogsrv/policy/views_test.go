package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestCreateView(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected apperrors.Error
	}{
		{
			name: "valid view",
			jsonData: `
		{
		    "apiVersion": "0.1.0-alpha.1",
		    "kind": "View",
		    "metadata": {
		        "name": "valid-view",
		        "catalog": "validcatalog",
		        "description": "This is a valid view"
		    },
		    "spec": {
		        "rules": [{
		            "intent": "Allow",
		            "actions": ["system.catalog.list"],
		            "targets": ["res://variants/my-variant/resources/coll-schema"]
		        }]
		    }
		}`,
			expected: nil,
		},
		{
			name: "valid view 2",
			jsonData: `
		{
		    "apiVersion": "0.1.0-alpha.1",
		    "kind": "View",
		    "metadata": {
		        "name": "valid-view2",
		        "catalog": "validcatalog",
		        "description": "This is a valid view"
		    },
		    "spec": {
		        "rules": [{
		            "intent": "Allow",
		            "actions": ["system.catalog.list"],
		            "targets": ["res://variants/my-variant/collectionschemas"]
		        }]
		    }
		}`,
			expected: nil,
		},
		{
			name: "valid view",
			jsonData: `
		{
		    "apiVersion": "0.1.0-alpha.1",
		    "kind": "View",
		    "metadata": {
		        "name": "valid-view3",
		        "catalog": "validcatalog",
		        "description": "This is a valid view"
		    },
		    "spec": {
		        "rules": [{
		            "intent": "Allow",
		            "actions": ["system.catalog.list"],
		            "targets": ["res://resources/*"]
		        }]
		    }
		}`,
			expected: nil,
		},
		{
			name: "valid view",
			jsonData: `
		{
		    "apiVersion": "0.1.0-alpha.1",
		    "kind": "View",
		    "metadata": {
		        "name": "valid-view4",
		        "catalog": "validcatalog",
		        "description": "This is a valid view"
		    },
		    "spec": {
		        "rules": [{
		            "intent": "Allow",
		            "actions": ["system.catalog.list"],
		            "targets": ["res://resources/*"]
		        }]
		    }
		}`,
			expected: nil,
		},
		{
			name: "empty rules",
			jsonData: `
				{
				    "apiVersion": "0.1.0-alpha.1",
				    "kind": "View",
				    "metadata": {
				        "name": "empty-rules-view",
				        "catalog": "validcatalog",
				        "description": "View with empty rules"
				    },
				    "spec": {
				        "rules": []
				    }
				}`,
			expected: ErrInvalidSchema,
		},
		{
			name: "invalid version",
			jsonData: `
				{
				    "apiVersion": "v2",
				    "kind": "View",
				    "metadata": {
				        "name": "invalid-version-view",
				        "catalog": "validcatalog",
				        "description": "Invalid version in view"
				    },
				    "spec": {
				        "rules": [{
				            "intent": "Allow",
				            "actions": ["system.catalog.list"],
				            "targets": ["res://catalogs/validcatalog"]
				        }]
				    }
				}`,
			expected: ErrInvalidSchema,
		},
		{
			name: "invalid kind",
			jsonData: `
				{
				    "apiVersion": "0.1.0-alpha.1",
				    "kind": "InvalidKind",
				    "metadata": {
				        "name": "invalid-kind-view",
				        "catalog": "validcatalog",
				        "description": "Invalid kind in view"
				    },
				    "spec": {
				        "rules": [{
				            "intent": "Allow",
				            "actions": ["system.catalog.list"],
				            "targets": ["res://catalogs/validcatalog"]
				        }]
				    }
				}`,
			expected: ErrInvalidSchema,
		},
		{
			name: "invalid name format",
			jsonData: `
				{
				    "apiVersion": "0.1.0-alpha.1",
				    "kind": "View",
				    "metadata": {
				        "name": "invalid name format",
				        "catalog": "validcatalog",
				        "description": "Invalid name format in view"
				    },
				    "spec": {
				        "rules": [{
				            "intent": "Allow",
				            "actions": ["system.catalog.list"],
				            "targets": ["res://catalogs/validcatalog"]
				        }]
				    }
				}`,
			expected: ErrInvalidSchema,
		},
		{
			name: "invalid rule effect",
			jsonData: `
				{
				    "apiVersion": "0.1.0-alpha.1",
				    "kind": "View",
				    "metadata": {
				        "name": "invalid-rule-effect",
				        "catalog": "validcatalog",
				        "description": "Invalid rule effect in view"
				    },
				    "spec": {
				        "rules": [{
				            "intent": "Invalid",
				            "actions": ["system.catalog.list"],
				            "targets": ["res://catalogs/validcatalog"]
				        }]
				    }
				}`,
			expected: ErrInvalidSchema,
		},
		{
			name: "invalid rule action",
			jsonData: `
				{
				    "apiVersion": "0.1.0-alpha.1",
				    "kind": "View",
				    "metadata": {
				        "name": "invalid-rule-action",
				        "catalog": "validcatalog",
				        "description": "Invalid rule action in view"
				    },
				    "spec": {
				        "rules": [{
				            "intent": "Allow",
				            "actions": ["system.Invalid"],
				            "targets": ["res://catalogs/validcatalog"]
				        }]
				    }
				}`,
			expected: ErrInvalidSchema,
		},
		{
			name: "invalid resource URI",
			jsonData: `
				{
				    "apiVersion": "0.1.0-alpha.1",
				    "kind": "View",
				    "metadata": {
				        "name": "invalid-resource-uri",
				        "catalog": "validcatalog",
				        "description": "Invalid resource URI in view"
				    },
				    "spec": {
				        "rules": [{
				            "intent": "Allow",
				            "actions": ["system.catalog.list"],
				            "targets": ["invalid-uri", "res://invalid-format", "res://InvalidCase"]
				        }]
				    }
				}`,
			expected: ErrInvalidSchema,
		},
		{
			name: "valid view with multiple actions",
			jsonData: `
				{
				    "apiVersion": "0.1.0-alpha.1",
				    "kind": "View",
				    "metadata": {
				        "name": "valid-view-multi-action",
				        "catalog": "validcatalog",
				        "description": "This is a valid view with multiple actions"
				    },
				    "spec": {
				        "rules": [{
				            "intent": "Allow",
				            "actions": ["system.catalog.list", "system.variant.list", "system.namespace.list"],
				            "targets": ["res://variants/my-variant"]
				        }]
				    }
				}`,
			expected: nil,
		},
		{
			name: "invalid rule action with mixed valid and invalid",
			jsonData: `
				{
				    "apiVersion": "0.1.0-alpha.1",
				    "kind": "View",
				    "metadata": {
				        "name": "invalid-mixed-actions",
				        "catalog": "validcatalog",
				        "description": "View with mixed valid and invalid actions"
				    },
				    "spec": {
				        "rules": [{
				            "intent": "Allow",
				            "actions": ["system.catalog.list", "system.InvalidAction", "system.variant.list"],
				            "targets": ["res://catalogs/validcatalog"]
				        }]
				    }
				}`,
			expected: ErrInvalidSchema,
		},
		{
			name: "deduplication of actions and resources",
			jsonData: `
				{
				    "apiVersion": "0.1.0-alpha.1",
				    "kind": "View",
				    "metadata": {
				        "name": "dedup-test-view",
				        "catalog": "validcatalog",
				        "description": "Test view for deduplication"
				    },
				    "spec": {
				        "rules": [{
				            "intent": "Allow",
				            "actions": ["system.catalog.list", "system.variant.list", "system.catalog.list", "system.namespace.list", "system.variant.list", "system.namespace.list"],
				            "targets": ["res://variants/my-variant"]
				        }]
				    }
				}`,
			expected: nil,
		},
		{
			name: "test with scopes",
			jsonData: `
		{
		    "apiVersion": "0.1.0-alpha.1",
		    "kind": "View",
		    "metadata": {
		        "name": "scope-override-test",
		        "catalog": "validcatalog",
		        "description": "Test view for deduplication"
		    },
		    "spec": {
		        "rules": [{
		            "intent": "Allow",
		            "actions": ["system.catalog.list", "system.variant.list", "system.catalog.list", "system.namespace.list", "system.variant.list", "system.namespace.list"],
		            "targets": ["res://variants/my-variant/resources/my-workspace"]
		        }]
		    }
		}`,
			expected: nil,
		},
		{
			name: "test with resource URI",
			jsonData: `
		{
		    "apiVersion": "0.1.0-alpha.1",
		    "kind": "View",
		    "metadata": {
		        "name": "test-with-resource-uri",
		        "catalog": "validcatalog",
		        "description": "Test view for deduplication"
		    },
		    "spec": {
		        "rules": [{
		            "intent": "Allow",
		            "actions": ["system.catalog.list", "system.variant.list", "system.catalog.list", "system.namespace.list", "system.variant.list", "system.namespace.list"],
		            "targets": ["res://resources/my-collection", "res://namespaces/my-namespace/resources/a/b/c/d"]
		        },
				{
		            "intent": "Allow",
		            "actions": ["system.catalog.list", "system.variant.list", "system.catalog.list", "system.namespace.list", "system.variant.list", "system.namespace.list"],
		            "targets": []
		        }]
		    }
		}`,
			expected: nil,
		},
		{
			name: "test with adopt view",
			jsonData: `
		{
		    "apiVersion": "0.1.0-alpha.1",
		    "kind": "View",
		    "metadata": {
		        "name": "test-with-adopt-view",
		        "catalog": "validcatalog",
		        "description": "Test view for deduplication"
		    },
		    "spec": {
		        "rules": [
					{
						"intent": "Allow",
						"actions": ["system.catalog.adoptView"],
						"targets": ["res://views/some-view/variants/test-variant"]
					}
				]
		    }
		}`,
			expected: nil,
		},
	}

	// Initialize context with logger and database connection
	ctx := newDb()
	defer db.DB(ctx).Close(ctx)

	tenantID, goerr := common.GetUniqueId(common.ID_TYPE_TENANT)
	require.NoError(t, goerr)
	projectID, goerr := common.GetUniqueId(common.ID_TYPE_PROJECT)
	require.NoError(t, goerr)

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, catcommon.TenantId(tenantID))
	ctx = catcommon.WithProjectID(ctx, catcommon.ProjectId(projectID))

	// Create the tenant and project for testing
	err := db.DB(ctx).CreateTenant(ctx, catcommon.TenantId(tenantID))
	require.NoError(t, err)
	defer db.DB(ctx).DeleteTenant(ctx, catcommon.TenantId(tenantID))

	err = db.DB(ctx).CreateProject(ctx, catcommon.ProjectId(projectID))
	require.NoError(t, err)
	defer db.DB(ctx).DeleteProject(ctx, catcommon.ProjectId(projectID))

	// Create a catalog for testing the variants
	catalogName := "validcatalog"
	err = db.DB(ctx).CreateCatalog(ctx, &models.Catalog{
		Name:        catalogName,
		Description: "Test catalog",
		ProjectID:   catcommon.ProjectId(projectID),
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)
	defer db.DB(ctx).DeleteCatalog(ctx, uuid.Nil, catalogName)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create metadata from the JSON data
			var viewData map[string]any
			err := json.Unmarshal([]byte(tt.jsonData), &viewData)
			require.NoError(t, err)

			metadata := &interfaces.Metadata{
				Catalog: viewData["metadata"].(map[string]any)["catalog"].(string),
			}

			_, err = CreateView(ctx, []byte(tt.jsonData), metadata)
			if tt.expected == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expected), "expected error to be %v", tt.expected)
			}
		})
	}
}

func TestUpdateView(t *testing.T) {
	ctx := newDb()
	defer db.DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	require.NoError(t, db.DB(ctx).CreateTenant(ctx, tenantID))
	defer db.DB(ctx).DeleteTenant(ctx, tenantID)

	require.NoError(t, db.DB(ctx).CreateProject(ctx, projectID))
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog first
	catalogID := uuid.New()
	err := db.DB(ctx).CreateCatalog(ctx, &models.Catalog{
		CatalogID:   catalogID,
		Name:        "test-catalog",
		Description: "Test catalog",
		ProjectID:   projectID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)

	// Create initial view
	initialView := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "test-view",
			"catalog": "test-catalog",
			"description": "Initial description"
		},
		"spec": {
			"rules": [
				{
					"intent": "Allow",
					"actions": ["system.catalog.list"],
					"targets": []
				}
			]
		}
	}`

	metadata := &interfaces.Metadata{
		Catalog: "test-catalog",
	}
	_, err = CreateView(ctx, []byte(initialView), metadata)
	require.NoError(t, err)

	// Test successful update with multiple actions
	updateView := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "test-view",
			"catalog": "test-catalog",
			"description": "Updated description"
		},
		"spec": {
			"rules": [
				{
					"intent": "Allow",
					"actions": ["system.catalog.list", "system.variant.list", "system.namespace.list"],
					"targets": []
				}
			]
		}
	}`

	_, err = UpdateView(ctx, []byte(updateView), metadata)
	require.NoError(t, err)

	// Verify the update
	retrieved, err := db.DB(ctx).GetViewByLabel(ctx, "test-view", catalogID)
	require.NoError(t, err)
	assert.Equal(t, "Updated description", retrieved.Description)

	// Verify the ViewDefinition structure
	var viewDef ViewDefinition
	jsonErr := json.Unmarshal(retrieved.Rules, &viewDef)
	require.NoError(t, jsonErr)

	// Verify scope
	assert.Equal(t, "test-catalog", viewDef.Scope.Catalog)
	assert.Equal(t, "", viewDef.Scope.Variant)
	assert.Equal(t, "", viewDef.Scope.Namespace)

	// Verify rules
	assert.Equal(t, 1, len(viewDef.Rules))
	assert.Equal(t, IntentAllow, viewDef.Rules[0].Intent)
	assert.ElementsMatch(t, []Action{ActionCatalogList, ActionVariantList, ActionNamespaceList}, viewDef.Rules[0].Actions)
	assert.Empty(t, viewDef.Rules[0].Targets)

	// Test updating non-existent view
	nonExistentView := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "non-existent-view",
			"catalog": "test-catalog",
			"description": "Should fail"
		},
		"spec": {
			"rules": [{
				"intent": "Allow",
				"actions": ["system.variant.list"],
				"targets": []
			}]
		}
	}`

	_, err = UpdateView(ctx, []byte(nonExistentView), metadata)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrViewNotFound))

	// Test updating with invalid catalog
	invalidCatalogView := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "test-view",
			"catalog": "non-existent-catalog",
			"description": "Should fail"
		},
		"spec": {
			"rules": [{
				"intent": "Allow",
				"actions": ["system.variant.list"],
				"targets": []
			}]
		}
	}`

	invalidMetadata := &interfaces.Metadata{
		Catalog: "non-existent-catalog",
	}
	_, err = UpdateView(ctx, []byte(invalidCatalogView), invalidMetadata)
	assert.Error(t, err)

	// Test updating with invalid JSON
	_, err = UpdateView(ctx, []byte("invalid json"), metadata)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidView))

	// Test updating with invalid schema
	invalidSchemaView := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "test-view",
			"catalog": "test-catalog"
		},
		"spec": {
			"rules": [
				{
					"intent": "InvalidEffect",
					"actions": ["system.catalog.list"],
					"targets": ["res://*"]
				}
			]
		}
	}`

	_, err = UpdateView(ctx, []byte(invalidSchemaView), metadata)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidSchema))

	// Test deduplication in update
	updateViewWithDuplicates := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "test-view",
			"catalog": "test-catalog",
			"description": "Updated description with duplicates"
		},
		"spec": {
			"rules": [
				{
					"intent": "Allow",
					"actions": ["system.catalog.list", "system.variant.list", "system.catalog.list", "system.namespace.list", "system.variant.list"],
					"targets": ["res://variants/valid-variant"]
				}
			]
		}
	}`

	_, err = UpdateView(ctx, []byte(updateViewWithDuplicates), metadata)
	require.NoError(t, err)

	// Verify the deduplication
	retrieved, err = db.DB(ctx).GetViewByLabel(ctx, "test-view", catalogID)
	require.NoError(t, err)

	var viewDef2 ViewDefinition
	jsonErr = json.Unmarshal(retrieved.Rules, &viewDef2)
	require.NoError(t, jsonErr)

	// Verify scope is preserved
	assert.Equal(t, "test-catalog", viewDef2.Scope.Catalog)
	assert.Equal(t, "", viewDef2.Scope.Variant)
	assert.Equal(t, "", viewDef2.Scope.Namespace)

	// Check that duplicates were removed
	assert.Equal(t, 1, len(viewDef2.Rules))
	assert.Equal(t, 3, len(viewDef2.Rules[0].Actions)) // Should have catalog.list, variant.list, namespace.list
	assert.Equal(t, 1, len(viewDef2.Rules[0].Targets)) // Should have one unique resource

	// Verify the order and content of deduplicated arrays
	expectedActions := []Action{ActionCatalogList, ActionVariantList, ActionNamespaceList}
	assert.ElementsMatch(t, expectedActions, viewDef2.Rules[0].Actions)

	expectedTargets := []TargetResource{"res://variants/valid-variant"}
	assert.ElementsMatch(t, expectedTargets, viewDef2.Rules[0].Targets)
}

func TestDeleteView(t *testing.T) {
	ctx := newDb()
	defer db.DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	require.NoError(t, db.DB(ctx).CreateTenant(ctx, tenantID))
	defer db.DB(ctx).DeleteTenant(ctx, tenantID)

	require.NoError(t, db.DB(ctx).CreateProject(ctx, projectID))
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog first
	catalogID := uuid.New()
	err := db.DB(ctx).CreateCatalog(ctx, &models.Catalog{
		CatalogID:   catalogID,
		Name:        "test-catalog",
		Description: "Test catalog",
		ProjectID:   projectID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)

	// Create test views
	testViews := []struct {
		name        string
		label       string
		description string
	}{
		{
			name:        "view1",
			label:       "test-view-1",
			description: "Test view 1",
		},
		{
			name:        "view2",
			label:       "test-view-2",
			description: "Test view 2",
		},
	}

	metadata := &interfaces.Metadata{
		Catalog: "test-catalog",
	}

	for _, tv := range testViews {
		view := `{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "View",
			"metadata": {
				"name": "%s",
				"catalog": "test-catalog",
				"description": "%s"
			},
			"spec": {
				"rules": [
					{
						"intent": "Allow",
						"actions": ["system.catalog.list"],
						"targets": ["res://variants/my-variant"]
					}
				]
			}
		}`
		viewJSON := fmt.Sprintf(view, tv.label, tv.description)
		_, err = CreateView(ctx, []byte(viewJSON), metadata)
		require.NoError(t, err)
	}

	t.Run("delete by label - success", func(t *testing.T) {
		// Delete first view by label
		reqCtx := interfaces.RequestContext{
			CatalogID:  catalogID,
			Catalog:    "test-catalog",
			ObjectName: "test-view-1",
		}
		vr, err := NewViewKindHandler(ctx, reqCtx)
		require.NoError(t, err)

		err = vr.Delete(ctx)
		assert.NoError(t, err)

		// Verify view is deleted
		_, err = db.DB(ctx).GetViewByLabel(ctx, "test-view-1", catalogID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, dberror.ErrNotFound))
	})

	t.Run("delete by label - non-existent view", func(t *testing.T) {
		reqCtx := interfaces.RequestContext{
			CatalogID:  catalogID,
			Catalog:    "test-catalog",
			ObjectName: "non-existent-view",
		}
		vr, err := NewViewKindHandler(ctx, reqCtx)
		require.NoError(t, err)

		err = vr.Delete(ctx)
		assert.NoError(t, err) // Should return nil for non-existent view
	})

	t.Run("delete by label - invalid catalog ID", func(t *testing.T) {
		reqCtx := interfaces.RequestContext{
			CatalogID:  uuid.Nil,
			Catalog:    "test-catalog",
			ObjectName: "test-view-2",
		}
		_, err := NewViewKindHandler(ctx, reqCtx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidCatalog))
	})

	t.Run("delete by label - empty label", func(t *testing.T) {
		reqCtx := interfaces.RequestContext{
			CatalogID:  catalogID,
			Catalog:    "test-catalog",
			ObjectName: "",
		}
		vr, err := NewViewKindHandler(ctx, reqCtx)
		require.NoError(t, err)

		err = vr.Delete(ctx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidView))
	})

	t.Run("delete by label - wrong catalog ID", func(t *testing.T) {
		wrongCatalogID := uuid.New()
		reqCtx := interfaces.RequestContext{
			CatalogID:  wrongCatalogID,
			Catalog:    "test-catalog",
			ObjectName: "test-view-2",
		}
		vr, err := NewViewKindHandler(ctx, reqCtx)
		require.NoError(t, err)

		err = vr.Delete(ctx)
		assert.NoError(t, err) // Should return nil as the view doesn't exist in this catalog
	})
}

func TestGetView(t *testing.T) {
	ctx := newDb()
	defer db.DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	require.NoError(t, db.DB(ctx).CreateTenant(ctx, tenantID))
	defer db.DB(ctx).DeleteTenant(ctx, tenantID)

	require.NoError(t, db.DB(ctx).CreateProject(ctx, projectID))
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog first
	catalogID := uuid.New()
	err := db.DB(ctx).CreateCatalog(ctx, &models.Catalog{
		CatalogID:   catalogID,
		Name:        "test-catalog",
		Description: "Test catalog",
		ProjectID:   projectID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)

	// Create test views
	testViews := []struct {
		name        string
		label       string
		description string
		actions     []string
		targets     []string
	}{
		{
			name:        "view1",
			label:       "test-view-1",
			description: "Test view 1",
			actions:     []string{"system.catalog.list", "system.variant.list"},
			targets:     []string{"res://variants/my-variant"},
		},
		{
			name:        "view2",
			label:       "test-view-2",
			description: "Test view 2",
			actions:     []string{"system.namespace.list"},
			targets:     []string{"res://namespaces/my-namespace"},
		},
	}

	metadata := &interfaces.Metadata{
		Catalog: "test-catalog",
	}

	for _, tv := range testViews {
		view := `{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "View",
			"metadata": {
				"name": "%s",
				"catalog": "test-catalog",
				"description": "%s"
			},
			"spec": {
				"rules": [
					{
						"intent": "Allow",
						"actions": %s,
						"targets": %s
					}
				]
			}
		}`
		actionsJSON, _ := json.Marshal(tv.actions)
		targetsJSON, _ := json.Marshal(tv.targets)
		viewJSON := fmt.Sprintf(view, tv.label, tv.description, string(actionsJSON), string(targetsJSON))
		_, err = CreateView(ctx, []byte(viewJSON), metadata)
		require.NoError(t, err)
	}

	t.Run("get by label - success", func(t *testing.T) {
		reqCtx := interfaces.RequestContext{
			CatalogID:  catalogID,
			Catalog:    "test-catalog",
			ObjectName: "test-view-1",
		}
		vr, err := NewViewKindHandler(ctx, reqCtx)
		require.NoError(t, err)

		viewJSON, err := vr.Get(ctx)
		require.NoError(t, err)

		// Verify the returned JSON
		var viewData map[string]any
		jsonErr := json.Unmarshal(viewJSON, &viewData)
		require.NoError(t, jsonErr)

		// Check metadata
		metadata := viewData["metadata"].(map[string]any)
		assert.Equal(t, "test-view-1", metadata["name"])
		assert.Equal(t, "test-catalog", metadata["catalog"])
		assert.Equal(t, "Test view 1", metadata["description"])

		// Check rules
		spec := viewData["spec"].(map[string]any)
		rules := spec["rules"].([]any)
		assert.Equal(t, 1, len(rules))

		rule := rules[0].(map[string]any)
		assert.Equal(t, "Allow", rule["intent"])
		assert.ElementsMatch(t, []any{"system.catalog.list", "system.variant.list"}, rule["actions"])
		assert.ElementsMatch(t, []any{"res://variants/my-variant"}, rule["targets"])
	})

	t.Run("get by label - non-existent view", func(t *testing.T) {
		reqCtx := interfaces.RequestContext{
			CatalogID:  catalogID,
			Catalog:    "test-catalog",
			ObjectName: "non-existent-view",
		}
		vr, err := NewViewKindHandler(ctx, reqCtx)
		require.NoError(t, err)

		_, err = vr.Get(ctx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrViewNotFound))
	})

	t.Run("get by label - invalid catalog ID", func(t *testing.T) {
		reqCtx := interfaces.RequestContext{
			CatalogID:  uuid.Nil,
			Catalog:    "test-catalog",
			ObjectName: "test-view-2",
		}
		_, err := NewViewKindHandler(ctx, reqCtx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidCatalog))
	})

	t.Run("get by label - empty label", func(t *testing.T) {
		reqCtx := interfaces.RequestContext{
			CatalogID:  catalogID,
			Catalog:    "test-catalog",
			ObjectName: "",
		}
		vr, err := NewViewKindHandler(ctx, reqCtx)
		require.NoError(t, err)

		_, err = vr.Get(ctx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidView))
	})

	t.Run("get by label - wrong catalog ID", func(t *testing.T) {
		wrongCatalogID := uuid.New()
		reqCtx := interfaces.RequestContext{
			CatalogID:  wrongCatalogID,
			Catalog:    "test-catalog",
			ObjectName: "test-view-2",
		}
		vr, err := NewViewKindHandler(ctx, reqCtx)
		require.NoError(t, err)

		_, err = vr.Get(ctx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrViewNotFound))
	})
}
