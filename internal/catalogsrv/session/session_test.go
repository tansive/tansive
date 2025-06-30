package session

import (
	"errors"
	"testing"

	"encoding/json"

	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	schemaerr "github.com/tansive/tansive/internal/catalogsrv/schema/errors"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestNewSession(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := newDb()
	defer db.DB(ctx).Close(ctx)

	// Create tenant and project
	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	require.NoError(t, db.DB(ctx).CreateTenant(ctx, tenantID))
	defer db.DB(ctx).DeleteTenant(ctx, tenantID)

	require.NoError(t, db.DB(ctx).CreateProject(ctx, projectID))
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog
	catalogID := uuid.New()
	err := db.DB(ctx).CreateCatalog(ctx, &models.Catalog{
		CatalogID:   catalogID,
		Name:        "test-catalog",
		Description: "Test catalog",
		ProjectID:   projectID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)
	defer db.DB(ctx).DeleteCatalog(ctx, uuid.Nil, "test-catalog")

	// Create a variant
	variantID := uuid.New()
	err = db.DB(ctx).CreateVariant(ctx, &models.Variant{
		VariantID:   variantID,
		Name:        "test-variant",
		Description: "Test variant",
		CatalogID:   catalogID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)

	// Set up catalog context
	ctx = catcommon.WithCatalogContext(ctx, &catcommon.CatalogContext{
		CatalogID: catalogID,
		Catalog:   "test-catalog",
		VariantID: variantID,
		Variant:   "test-variant",
		UserContext: &catcommon.UserContext{
			UserID: "users/testuser",
		},
	})

	// Create a skillset
	skillsetJson := []byte(`{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "SkillSet",
		"metadata": {
			"name": "test-skillset",
			"description": "Test skillset",
			"catalog": "test-catalog",
			"variant": "test-variant",
			"path": "/skills"
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
						"script": "test.py"
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
				},
				{
					"name": "python-skill",
					"description": "Python skill",
					"source": "python-runner",
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
					"exportedActions": ["python.action"]
				}
			],
			"dependencies": [
				{
					"path": "/resources/test",
					"kind": "Resource",
					"alias": "test-resource",
					"actions": ["read"],
					"export": false
				}
			]
		}
	}`)

	// Create and save the skillset
	sm, err := catalogmanager.NewSkillSetManager(ctx, skillsetJson, nil)
	require.NoError(t, err)
	err = sm.Save(ctx)
	require.NoError(t, err)

	// Create a parent view
	parentView := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "parent-view",
			"catalog": "test-catalog",
			"variant": "test-variant",
			"description": "Parent view for testing"
		},
		"spec": {
			"rules": [{
				"intent": "Allow",
				"actions": ["system.catalog.list", "system.variant.list", "system.namespace.list", "test.action"],
				"targets": ["res://*"]
			},
			{
				"intent": "Allow",
				"actions": ["system.catalog.adoptView"],
				"targets": ["res://views/parent-view"]
			},
			{
				"intent": "Allow",
				"actions": ["system.skillset.use"],
				"targets": ["res://skillsets/skills/test-skillset"]
			}]
		}
	}`

	metadata := &interfaces.Metadata{
		Catalog: "test-catalog",
	}
	view, err := policy.CreateView(ctx, []byte(parentView), metadata)
	require.NoError(t, err)

	// Set the parent view in context
	var viewDef policy.ViewDefinition
	if err := json.Unmarshal(view.Rules, &viewDef); err != nil {
		t.Fatalf("failed to unmarshal view definition: %v", err)
	}
	ctx = policy.WithViewDefinition(ctx, &viewDef)

	tests := []struct {
		name        string
		sessionSpec string
		wantErr     bool
		errType     error
	}{
		{
			name: "valid session spec",
			sessionSpec: `{
				"skillPath": "/skills/test-skillset/test-skill",
				"viewName": "parent-view",
				"sessionVariables": {
					"key1": "value1",
					"key2": 123,
					"key3": true
				},
				"inputArgs": {
					"input": "test input",
					"params": {
						"param1": "value1",
						"param2": 42
					}
				}
			}`,
			wantErr: false,
		},
		{
			name: "missing skillPath",
			sessionSpec: `{
				"viewName": "parent-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantErr: true,
			errType: ErrInvalidSession,
		},
		{
			name: "missing viewName",
			sessionSpec: `{
				"skillPath": "/skills/test-skillset/test-skill",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantErr: true,
			errType: ErrInvalidSession,
		},
		{
			name: "invalid skillPath format",
			sessionSpec: `{
				"skillPath": "invalid/path/format",
				"viewName": "parent-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantErr: true,
			errType: ErrInvalidSession,
		},
		{
			name: "invalid viewName format",
			sessionSpec: `{
				"skillPath": "/skills/test-skillset/test-skill",
				"viewName": "invalid view name",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantErr: true,
			errType: ErrInvalidSession,
		},
		{
			name: "too many variables",
			sessionSpec: `{
				"skillPath": "/skills/test-skillset/test-skill",
				"viewName": "parent-view",
				"sessionVariables": {
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
					"key4": "value4",
					"key5": "value5",
					"key6": "value6",
					"key7": "value7",
					"key8": "value8",
					"key9": "value9",
					"key10": "value10",
					"key11": "value11",
					"key12": "value12",
					"key13": "value13",
					"key14": "value14",
					"key15": "value15",
					"key16": "value16",
					"key17": "value17",
					"key18": "value18",
					"key19": "value19",
					"key20": "value20",
					"key21": "value21"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantErr: true,
			errType: ErrInvalidSession,
		},
		{
			name: "invalid variable key format",
			sessionSpec: `{
				"skillPath": "/skills/test-skillset/test-skill",
				"viewName": "parent-view",
				"sessionVariables": {
					"invalid@key": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantErr: true,
			errType: ErrInvalidSession,
		},
		{
			name: "invalid variable value type",
			sessionSpec: `{
				"skillPath": "/skills/test-skillset/test-skill",
				"viewName": "parent-view",
				"sessionVariables": {
					"validKey": {"invalid": "object"}
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantErr: false,
			errType: ErrInvalidSession,
		},
		{
			name: "non-existent view",
			sessionSpec: `{
				"skillPath": "/skills/test-skillset/test-skill",
				"viewName": "non-existent-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantErr: true,
			errType: ErrDisallowedByPolicy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := NewSession(ctx, []byte(tt.sessionSpec))
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					if tt.errType == ErrInvalidSession {
						// For validation errors, check if it's a schema validation error
						var validationErrors schemaerr.ValidationErrors
						if errors.As(err, &validationErrors) {
							assert.NotEmpty(t, validationErrors)
						} else {
							// If it's not a validation error, it should be ErrInvalidSession
							assert.True(t, errors.Is(err, tt.errType), "expected error to be %v, got %v", tt.errType, err)
						}
					} else {
						assert.True(t, errors.Is(err, tt.errType), "expected error to be %v, got %v", tt.errType, err)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSessionSpec_Validate(t *testing.T) {
	config.TestInit()
	Init()
	tests := []struct {
		name    string
		spec    SessionSpec
		wantErr bool
	}{
		{
			name: "valid spec",
			spec: SessionSpec{
				SkillPath:        "/skills/test-skill",
				ViewName:         "test-view",
				SessionVariables: json.RawMessage(`{"key1": "value1"}`),
			},
			wantErr: false,
		},
		{
			name: "missing skillPath",
			spec: SessionSpec{
				ViewName:         "test-view",
				SessionVariables: json.RawMessage(`{"key1": "value1"}`),
			},
			wantErr: true,
		},
		{
			name: "missing viewName",
			spec: SessionSpec{
				SkillPath:        "skills/test-skill",
				SessionVariables: json.RawMessage(`{"key1": "value1"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid skillPath format",
			spec: SessionSpec{
				SkillPath:        "invalid/path/format",
				ViewName:         "test-view",
				SessionVariables: json.RawMessage(`{"key1": "value1"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid viewName format",
			spec: SessionSpec{
				SkillPath:        "skills/test-skill",
				ViewName:         "invalid view name",
				SessionVariables: json.RawMessage(`{"key1": "value1"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid variables format",
			spec: SessionSpec{
				SkillPath:        "skills/test-skill",
				ViewName:         "test-view",
				SessionVariables: json.RawMessage(`invalid json`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Empty(t, err)
			}
		})
	}
}

func marshalJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	jsonBytes, goerr := json.Marshal(v)
	if goerr != nil {
		t.Fatalf("failed to marshal JSON: %v", goerr)
	}
	return jsonBytes
}

func TestSessionSaveAndGet(t *testing.T) {
	config.TestInit()
	Init()
	// Initialize context with logger and database connection
	ctx := newDb()
	defer db.DB(ctx).Close(ctx)

	// Create tenant and project
	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	require.NoError(t, db.DB(ctx).CreateTenant(ctx, tenantID))
	defer db.DB(ctx).DeleteTenant(ctx, tenantID)

	require.NoError(t, db.DB(ctx).CreateProject(ctx, projectID))
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog
	catalogID := uuid.New()
	err := db.DB(ctx).CreateCatalog(ctx, &models.Catalog{
		CatalogID:   catalogID,
		Name:        "test-catalog",
		Description: "Test catalog",
		ProjectID:   projectID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)
	defer db.DB(ctx).DeleteCatalog(ctx, uuid.Nil, "test-catalog")

	// Create a variant
	variantID := uuid.New()
	err = db.DB(ctx).CreateVariant(ctx, &models.Variant{
		VariantID:   variantID,
		Name:        "test-variant",
		Description: "Test variant",
		CatalogID:   catalogID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)

	// Set up catalog context
	ctx = catcommon.WithCatalogContext(ctx, &catcommon.CatalogContext{
		CatalogID: catalogID,
		Catalog:   "test-catalog",
		VariantID: variantID,
		Variant:   "test-variant",
		UserContext: &catcommon.UserContext{
			UserID: "users/testuser",
		},
	})

	// Create a skillset
	skillsetJson := []byte(`{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "SkillSet",
		"metadata": {
			"name": "test-skillset",
			"description": "Test skillset",
			"catalog": "test-catalog",
			"variant": "test-variant",
			"path": "/skills"
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
						"script": "test.py"
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
				},
				{
					"name": "python-skill",
					"description": "Python skill",
					"source": "python-runner",
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
					"exportedActions": ["python.action"]
				}
			],
			"dependencies": [
				{
					"path": "/resources/test",
					"kind": "Resource",
					"alias": "test-resource",
					"actions": ["read"],
					"export": false
				}
			]
		}
	}`)

	// Create and save the skillset
	sm, err := catalogmanager.NewSkillSetManager(ctx, skillsetJson, nil)
	require.NoError(t, err)
	err = sm.Save(ctx)
	require.NoError(t, err)

	// Create a parent view
	parentView := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "parent-view",
			"catalog": "test-catalog",
			"variant": "test-variant",
			"description": "Parent view for testing"
		},
		"spec": {
			"rules": [{
				"intent": "Allow",
				"actions": ["system.catalog.list", "system.variant.list", "system.namespace.list", "test.action"],
				"targets": ["res://*"]
			},
			{
				"intent": "Allow",
				"actions": ["system.catalog.adoptView"],
				"targets": ["res://views/parent-view"]
			},
			{
				"intent": "Allow",
				"actions": ["system.skillset.use"],
				"targets": ["res://skillsets/skills/test-skillset"]
			}]
		}
	}`

	metadata := &interfaces.Metadata{
		Catalog: "test-catalog",
	}
	view, err := policy.CreateView(ctx, []byte(parentView), metadata)
	require.NoError(t, err)

	// Set the parent view in context
	var viewDef policy.ViewDefinition
	if err := json.Unmarshal(view.Rules, &viewDef); err != nil {
		t.Fatalf("failed to unmarshal view definition: %v", err)
	}
	ctx = policy.WithViewDefinition(ctx, &viewDef)

	// Test cases
	tests := []struct {
		name        string
		sessionSpec string
		wantErr     bool
	}{
		{
			name: "valid session save and get",
			sessionSpec: `{
				"skillPath": "/skills/test-skillset/test-skill",
				"viewName": "parent-view",
				"sessionVariables": {
					"key1": "value1",
					"key2": 123,
					"key3": true
				},
				"inputArgs": {
					"input": "test input",
					"params": {
						"param1": "value1",
						"param2": 42
					}
				}
			}`,
			wantErr: false,
		},
		{
			name: "session with empty variables",
			sessionSpec: `{
				"skillPath": "/skills/test-skillset/test-skill",
				"viewName": "parent-view",
				"sessionVariables": {},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create and save session
			session, _, err := NewSession(ctx, []byte(tt.sessionSpec))
			require.NoError(t, err)

			err = session.Save(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Get session by ID
			gotSession, err := GetSession(ctx, session.(*sessionManager).session.SessionID)
			assert.NoError(t, err)
			assert.NotNil(t, gotSession)

			// Verify session details
			originalSession := session.(*sessionManager).session
			retrievedSession := gotSession.(*sessionManager).session

			assert.Equal(t, originalSession.SessionID, retrievedSession.SessionID)
			assert.Equal(t, originalSession.SkillSet, retrievedSession.SkillSet)
			assert.Equal(t, originalSession.Skill, retrievedSession.Skill)
			assert.Equal(t, originalSession.ViewID, retrievedSession.ViewID)

			// Compare parsed JSON values instead of raw bytes
			var originalInfo, retrievedInfo SessionInfo
			if goerr := json.Unmarshal(originalSession.Info, &originalInfo); goerr != nil {
				t.Fatalf("failed to unmarshal original info: %v", goerr)
			}
			if goerr := json.Unmarshal(retrievedSession.Info, &retrievedInfo); goerr != nil {
				t.Fatalf("failed to unmarshal retrieved info: %v", goerr)
			}

			// Compare each field of SessionInfo
			originalVarsJSON := marshalJSON(t, originalInfo.SessionVariables)
			retrievedVarsJSON := marshalJSON(t, retrievedInfo.SessionVariables)
			assert.JSONEq(t, string(originalVarsJSON), string(retrievedVarsJSON))

			originalInputJSON := marshalJSON(t, originalInfo.InputArgs)
			retrievedInputJSON := marshalJSON(t, retrievedInfo.InputArgs)
			assert.JSONEq(t, string(originalInputJSON), string(retrievedInputJSON))

			originalViewJSON := marshalJSON(t, originalInfo.ViewDefinition)
			retrievedViewJSON := marshalJSON(t, retrievedInfo.ViewDefinition)
			assert.JSONEq(t, string(originalViewJSON), string(retrievedViewJSON))

			assert.Equal(t, originalSession.StatusSummary, retrievedSession.StatusSummary)
			assert.Equal(t, originalSession.UserID, retrievedSession.UserID)
			assert.Equal(t, originalSession.CatalogID, retrievedSession.CatalogID)
			assert.Equal(t, originalSession.VariantID, retrievedSession.VariantID)
		})
	}

	// Test error cases
	t.Run("get non-existent session", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := GetSession(ctx, nonExistentID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidObject))
	})

	// Test upsert functionality
	t.Run("session upsert updates existing session", func(t *testing.T) {
		// Create initial session
		initialSpec := `{
			"skillPath": "/skills/test-skillset/test-skill",
			"viewName": "parent-view",
			"sessionVariables": {
				"key1": "initial_value"
			},
			"inputArgs": {
				"input": "initial_input"
			}
		}`
		session, _, err := NewSession(ctx, []byte(initialSpec))
		require.NoError(t, err)
		err = session.Save(ctx)
		require.NoError(t, err)

		// Get the session ID
		sessionID := session.(*sessionManager).session.SessionID

		// Create a new session with the same ID but different values
		updatedSpec := `{
			"skillPath": "/skills/test-skillset/test-skill",
			"viewName": "parent-view",
			"sessionVariables": {
				"key1": "updated_value",
				"key2": "new_value"
			},
			"inputArgs": {
				"input": "updated_input",
				"new_param": "new_value"
			}
		}`
		updatedSession, _, err := NewSession(ctx, []byte(updatedSpec))
		require.NoError(t, err)
		updatedSession.(*sessionManager).session.SessionID = sessionID // Force same ID
		err = updatedSession.Save(ctx)
		require.NoError(t, err)

		// Retrieve the session and verify it was updated
		retrievedSession, err := GetSession(ctx, sessionID)
		require.NoError(t, err)
		require.NotNil(t, retrievedSession)

		// Verify the session was updated with new values
		var retrievedInfo SessionInfo
		if goerr := json.Unmarshal(retrievedSession.(*sessionManager).session.Info, &retrievedInfo); goerr != nil {
			t.Fatalf("failed to unmarshal retrieved info: %v", goerr)
		}

		// Verify session variables were updated
		assert.Equal(t, "updated_value", retrievedInfo.SessionVariables["key1"])
		assert.Equal(t, "new_value", retrievedInfo.SessionVariables["key2"])

		// Verify input args were updated
		assert.Equal(t, "updated_input", retrievedInfo.InputArgs["input"])
		assert.Equal(t, "new_value", retrievedInfo.InputArgs["new_param"])
	})
}
