package server

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/session"
	"github.com/tansive/tansive/internal/catalogsrv/tangent"
	"github.com/tansive/tansive/internal/common/uuid"
)

func generateTangentKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(nil)
}

func signRequest(r *http.Request, privateKey ed25519.PrivateKey, tangentID uuid.UUID) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	var body []byte
	var err error

	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		r.Body = io.NopCloser(strings.NewReader(string(body)))
	}

	stringToSign := strings.Join([]string{
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		string(body),
		timestamp,
	}, "\n")

	signature := ed25519.Sign(privateKey, []byte(stringToSign))
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	r.Header.Set("X-Tangent-Signature", signatureB64)
	r.Header.Set("X-Tangent-Signature-Timestamp", timestamp)
	r.Header.Set("X-TangentID", tangentID.String())

	return nil
}

func TestSessionCrud(t *testing.T) {
	ctx := newDb()
	t.Cleanup(func() {
		db.DB(ctx).Close(ctx)
	})

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("PABCDE")

	config.Config().DefaultProjectID = string(projectID)
	config.Config().DefaultTenantID = string(tenantID)

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant for testing
	err := db.DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = db.DB(ctx).DeleteTenant(ctx, tenantID)
	})

	// Create the project for testing
	err = db.DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	testContext := TestContext{
		TenantId:       tenantID,
		ProjectId:      projectID,
		CatalogContext: catcommon.CatalogContext{},
	}

	// Create a catalog
	httpReq, _ := http.NewRequest("POST", "/catalogs", nil)
	req := `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Catalog",
			"metadata": {
				"name": "valid-catalog",
				"description": "This is a valid catalog"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	response := executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	testContext.CatalogContext.Catalog = "valid-catalog"

	// Create a variant
	httpReq, _ = http.NewRequest("POST", "/variants", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Variant",
			"metadata": {
				"name": "valid-variant",
				"description": "This is a valid variant"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	testContext.CatalogContext.Variant = "valid-variant"

	// Create a skillset
	httpReq, _ = http.NewRequest("POST", "/skillsets", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "SkillSet",
			"metadata": {
				"name": "valid-skillset",
				"catalog": "valid-catalog",
				"variant": "valid-variant",
				"namespace": "",
				"path": "/",
				"description": "This is a valid skillset"
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
				]
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Create a view
	httpReq, _ = http.NewRequest("POST", "/views", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "View",
			"metadata": {
				"name": "valid-view",
				"catalog": "valid-catalog",
				"variant": "valid-variant",
				"description": "This is a valid view"
			},
			"spec": {
				"rules": [
					{
						"intent": "Allow",
						"actions": ["test.action"],
						"targets": ["res://*"]
					},
					{
						"intent": "Allow",
						"actions": ["system.catalog.adoptView"],
						"targets": ["res://views/valid-view"]
					},
					{
						"intent": "Allow",
						"actions": ["system.skillset.use"],
						"targets": ["res://skillsets/valid-skillset"]
					}
				]
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Create a session
	httpReq, _ = http.NewRequest("POST", "/sessions", nil)
	req = `
		{
			"skillPath": "/valid-skillset/test-skill",
			"viewName": "valid-view",
			"sessionVariables": {
				"key1": "value1",
				"key2": 123,
				"key3": true
			},
			"inputArgs": {
				"input": "test input"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Extract session ID from Location header
	location := response.Header().Get("Location")
	require.Equal(t, http.StatusCreated, response.Code)
	require.NotEmpty(t, location)

	// Test execution state creation and retrieval
	t.Run("execution state flow", func(t *testing.T) {
		// Create a tangent with Ed25519 keys for signing
		publicKey, privateKey, err := generateTangentKeyPair()
		require.NoError(t, err)

		tangentID := uuid.New()
		tangentInfo := &tangent.TangentInfo{
			ID:                     tangentID,
			CreatedBy:              "test-user",
			URL:                    "http://test.tansive.dev:8468",
			Capabilities:           []catcommon.RunnerID{"system.commandrunner"},
			PublicKeyAccessKey:     publicKey,
			PublicKeyLogSigningKey: []byte("test-signing-key"),
		}

		// Create tangent in database
		infoBytes, err := json.Marshal(tangentInfo)
		require.NoError(t, err)

		tangentModel := &models.Tangent{
			ID:        tangentID,
			Info:      infoBytes,
			PublicKey: publicKey,
			Status:    "active",
		}

		err = db.DB(ctx).CreateTangent(ctx, tangentModel)
		require.NoError(t, err)

		codeVerifier := "test_challenge"
		hashed := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hashed[:])

		// First create an interactive session
		httpReq, _ := http.NewRequest("POST", "/sessions?interactive=true&code_challenge="+codeChallenge, nil)
		req := `
			{
				"skillPath": "/valid-skillset/test-skill",
				"viewName": "valid-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test input"
				}
			}`
		setRequestBodyAndHeader(t, httpReq, req)
		response := executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusOK, response.Code)

		var sessionResp session.InteractiveSessionRsp
		err = json.Unmarshal(response.Body.Bytes(), &sessionResp)
		assert.NoError(t, err)
		assert.NotEmpty(t, sessionResp.Code)

		// Create execution state with the code - this uses tangentAuthMiddleware
		newCtx := newDb()
		t.Cleanup(func() {
			db.DB(newCtx).Close(newCtx)
		})

		httpReq, _ = http.NewRequest("POST", "/sessions/execution-state?code="+sessionResp.Code+"&code_verifier="+codeVerifier, nil)
		// Sign the request with tangent credentials
		err = signRequest(httpReq, privateKey, tangentID)
		require.NoError(t, err)
		response = executeTestRequest(t, httpReq, nil)
		require.Equal(t, http.StatusOK, response.Code)

		var tokenResp session.SessionTokenRsp
		err = json.Unmarshal(response.Body.Bytes(), &tokenResp)
		assert.NoError(t, err)

		// Get execution state to get the session ID
		httpReq, _ = http.NewRequest("GET", "/sessions/execution-state", nil)
		httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
		response = executeTestRequest(t, httpReq, nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var executionState session.ExecutionState
		err = json.Unmarshal(response.Body.Bytes(), &executionState)
		assert.NoError(t, err)

		// Update execution state - this uses tangentAuthMiddleware
		httpReq, _ = http.NewRequest("PUT", "/sessions/execution-state", nil)
		httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
		httpReq.Header.Set("Content-Type", "application/json")
		updateReq := `
			{
				"sessionID": "` + executionState.SessionID.String() + `",
				"statusSummary": "completed",
				"status": {
					"error": {
						"message": "test error",
						"code": "TEST_ERROR"
					},
					"auditLog": "test audit log"
				}
			}`
		setRequestBodyAndHeader(t, httpReq, updateReq)
		// Sign the request with tangent credentials
		err = signRequest(httpReq, privateKey, tangentID)
		require.NoError(t, err)
		response = executeTestRequest(t, httpReq, nil)
		require.Equal(t, http.StatusOK, response.Code)

		// Verify the update by getting session summary
		httpReq, _ = http.NewRequest("GET", "/sessions/summary?sessionID="+executionState.SessionID.String(), nil)
		httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
		response = executeTestRequest(t, httpReq, nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var summary session.SessionSummaryInfo
		err = json.Unmarshal(response.Body.Bytes(), &summary)
		assert.NoError(t, err)
		assert.Equal(t, session.SessionStatusCompleted, summary.StatusSummary)
		assert.Equal(t, "test error", summary.Error["message"])
		assert.Equal(t, "TEST_ERROR", summary.Error["code"])

		// Test error cases for updateExecutionState
		t.Run("update execution state error cases", func(t *testing.T) {
			tests := []struct {
				name       string
				body       string
				wantStatus int
			}{
				{
					name:       "missing body",
					body:       "",
					wantStatus: http.StatusBadRequest,
				},
				{
					name:       "invalid json",
					body:       "{invalid json}",
					wantStatus: http.StatusBadRequest,
				},
				{
					name: "invalid error format",
					body: `{
						"sessionID": "` + executionState.SessionID.String() + `",
						"statusSummary": "completed",
						"status": {
							"error": "invalid error format",
							"auditLog": "test audit log"
						}
					}`,
					wantStatus: http.StatusBadRequest,
				},
				{
					name: "invalid status summary",
					body: `{
						"sessionID": "` + executionState.SessionID.String() + `",
						"statusSummary": "invalid_status",
						"status": {
							"error": {
								"message": "test error",
								"code": "TEST_ERROR"
							},
							"auditLog": "test audit log"
							"auditLogVerificationKey": "test audit log verification key"
						}
					}`,
					wantStatus: http.StatusBadRequest,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					httpReq, _ := http.NewRequest("PUT", "/sessions/execution-state", nil)
					httpReq.Header.Set("Content-Type", "application/json")
					if tt.body != "" {
						setRequestBodyAndHeader(t, httpReq, tt.body)
					}
					// Sign the request with tangent credentials
					err = signRequest(httpReq, privateKey, tangentID)
					require.NoError(t, err)
					response := executeTestRequest(t, httpReq, nil)
					assert.Equal(t, tt.wantStatus, response.Code)
				})
			}
		})
	})

	// // Test interactive session creation
	t.Run("interactive session creation", func(t *testing.T) {
		config.Config().DefaultTenantID = string(tenantID)
		config.Config().DefaultProjectID = string(projectID)

		httpReq, _ := http.NewRequest("POST", "/sessions?interactive=true&code_challenge=test_challenge", nil)
		req := `
			{
				"skillPath": "/valid-skillset/test-skill",
				"viewName": "valid-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test input"
				}
			}`
		setRequestBodyAndHeader(t, httpReq, req)
		response := executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusOK, response.Code)

		var sessionResp session.InteractiveSessionRsp
		err := json.Unmarshal(response.Body.Bytes(), &sessionResp)
		assert.NoError(t, err)
		assert.NotEmpty(t, sessionResp.Code)
		assert.NotEmpty(t, sessionResp.TangentURL)
	})

	// // Test interactive session without code challenge
	t.Run("interactive session without code challenge", func(t *testing.T) {
		httpReq, _ := http.NewRequest("POST", "/sessions?interactive=true", nil)
		req := `
			{
				"skillPath": "/valid-skillset/test-skill",
				"viewName": "valid-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test input"
				}
			}`
		setRequestBodyAndHeader(t, httpReq, req)
		response := executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	// // Test invalid session creation
	tests := []struct {
		name        string
		sessionSpec string
		wantStatus  int
	}{
		{
			name: "missing skillPath",
			sessionSpec: `{
				"viewName": "valid-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing viewName",
			sessionSpec: `{
				"skillPath": "/valid-skillset/test-skill",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid skillPath",
			sessionSpec: `{
				"skillPath": "invalid/path",
				"viewName": "valid-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "non-existent view",
			sessionSpec: `{
				"skillPath": "/valid-skillset/test-skill",
				"viewName": "non-existent-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test"
				}
			}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Config().DefaultTenantID = string(tenantID)
			config.Config().DefaultProjectID = string(projectID)
			httpReq, _ := http.NewRequest("POST", "/sessions", nil)
			setRequestBodyAndHeader(t, httpReq, tt.sessionSpec)
			response := executeTestRequest(t, httpReq, nil, testContext)
			assert.Equal(t, tt.wantStatus, response.Code)
		})
	}

	// Test getSessions API
	t.Run("get sessions", func(t *testing.T) {
		httpReq, _ := http.NewRequest("GET", "/sessions", nil)
		httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
		response := executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusOK, response.Code)

		var sessions []session.SessionSummaryInfo
		err := json.Unmarshal(response.Body.Bytes(), &sessions)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(sessions), 1) // Should have at least one session from previous tests
	})

	// Test getSessionSummaryByID API
	t.Run("get session summary by ID", func(t *testing.T) {
		// First create a session to get its ID
		httpReq, _ := http.NewRequest("POST", "/sessions", nil)
		req := `
			{
				"skillPath": "/valid-skillset/test-skill",
				"viewName": "valid-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test input"
				}
			}`
		setRequestBodyAndHeader(t, httpReq, req)
		response := executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusCreated, response.Code)

		// Extract session ID from Location header
		location := response.Header().Get("Location")
		require.NotEmpty(t, location)
		sessionID := location[strings.LastIndex(location, "/")+1:]

		// Now get the session summary
		httpReq, _ = http.NewRequest("GET", "/sessions/summary?sessionID="+sessionID, nil)
		httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
		response = executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusOK, response.Code)

		var summary session.SessionSummaryInfo
		err := json.Unmarshal(response.Body.Bytes(), &summary)
		assert.NoError(t, err)
		assert.Equal(t, sessionID, summary.SessionID.String())
		assert.Equal(t, session.SessionStatusCreated, summary.StatusSummary)
	})

	// Test updateExecutionState API
	t.Run("update execution state", func(t *testing.T) {
		// Create a tangent with Ed25519 keys for signing
		publicKey, privateKey, err := generateTangentKeyPair()
		require.NoError(t, err)

		tangentID := uuid.New()
		tangentInfo := &tangent.TangentInfo{
			ID:                     tangentID,
			CreatedBy:              "test-user",
			URL:                    "http://test.tansive.dev:8468",
			Capabilities:           []catcommon.RunnerID{"system.commandrunner"},
			PublicKeyAccessKey:     publicKey,
			PublicKeyLogSigningKey: []byte("test-signing-key"),
		}

		// Create tangent in database
		infoBytes, err := json.Marshal(tangentInfo)
		require.NoError(t, err)

		tangentModel := &models.Tangent{
			ID:        tangentID,
			Info:      infoBytes,
			PublicKey: publicKey,
			Status:    "active",
		}

		err = db.DB(ctx).CreateTangent(ctx, tangentModel)
		require.NoError(t, err)

		// First create a session to get its ID
		httpReq, _ := http.NewRequest("POST", "/sessions", nil)
		req := `
			{
				"skillPath": "/valid-skillset/test-skill",
				"viewName": "valid-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test input"
				}
			}`
		setRequestBodyAndHeader(t, httpReq, req)
		response := executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusCreated, response.Code)

		// Create execution state with code verifier
		codeVerifier := "test_challenge"
		hashed := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hashed[:])

		// Create interactive session
		httpReq, _ = http.NewRequest("POST", "/sessions?interactive=true&code_challenge="+codeChallenge, nil)
		setRequestBodyAndHeader(t, httpReq, req)
		response = executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusOK, response.Code)

		var sessionResp session.InteractiveSessionRsp
		err = json.Unmarshal(response.Body.Bytes(), &sessionResp)
		assert.NoError(t, err)

		// Create execution state - this uses tangentAuthMiddleware
		httpReq, _ = http.NewRequest("POST", "/sessions/execution-state?code="+sessionResp.Code+"&code_verifier="+codeVerifier, nil)
		// Sign the request with tangent credentials
		err = signRequest(httpReq, privateKey, tangentID)
		require.NoError(t, err)
		response = executeTestRequest(t, httpReq, nil)
		require.Equal(t, http.StatusOK, response.Code)

		var tokenResp session.SessionTokenRsp
		err = json.Unmarshal(response.Body.Bytes(), &tokenResp)
		assert.NoError(t, err)

		// Get execution state to get the session ID
		httpReq, _ = http.NewRequest("GET", "/sessions/execution-state", nil)
		httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
		response = executeTestRequest(t, httpReq, nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var executionState session.ExecutionState
		err = json.Unmarshal(response.Body.Bytes(), &executionState)
		assert.NoError(t, err)

		// Update execution state - this uses tangentAuthMiddleware
		httpReq, _ = http.NewRequest("PUT", "/sessions/execution-state", nil)
		httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
		httpReq.Header.Set("Content-Type", "application/json")
		updateReq := `
			{
				"sessionID": "` + executionState.SessionID.String() + `",
				"statusSummary": "completed",
				"status": {
					"error": {
						"message": "test error",
						"code": "TEST_ERROR"
					},
					"auditLog": "test audit log"
				}
			}`
		setRequestBodyAndHeader(t, httpReq, updateReq)
		// Sign the request with tangent credentials
		err = signRequest(httpReq, privateKey, tangentID)
		require.NoError(t, err)
		response = executeTestRequest(t, httpReq, nil)
		assert.Equal(t, http.StatusOK, response.Code)

		// Verify the update by getting session summary
		httpReq, _ = http.NewRequest("GET", "/sessions/summary?sessionID="+executionState.SessionID.String(), nil)
		httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
		response = executeTestRequest(t, httpReq, nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var summary session.SessionSummaryInfo
		err = json.Unmarshal(response.Body.Bytes(), &summary)
		assert.NoError(t, err)
		assert.Equal(t, session.SessionStatusCompleted, summary.StatusSummary)
		assert.Equal(t, "test error", summary.Error["message"])
		assert.Equal(t, "TEST_ERROR", summary.Error["code"])
	})

	// Test getAuditLogVerificationKeyByID API
	t.Run("get audit log verification key by ID", func(t *testing.T) {
		// Create a tangent with Ed25519 keys for signing
		publicKey, privateKey, err := generateTangentKeyPair()
		require.NoError(t, err)

		tangentID := uuid.New()
		tangentInfo := &tangent.TangentInfo{
			ID:                     tangentID,
			CreatedBy:              "test-user",
			URL:                    "http://test.tansive.dev:8468",
			Capabilities:           []catcommon.RunnerID{"system.commandrunner"},
			PublicKeyAccessKey:     publicKey,
			PublicKeyLogSigningKey: []byte("test-signing-key"),
		}

		// Create tangent in database
		infoBytes, err := json.Marshal(tangentInfo)
		require.NoError(t, err)

		tangentModel := &models.Tangent{
			ID:        tangentID,
			Info:      infoBytes,
			PublicKey: publicKey,
			Status:    "active",
		}

		err = db.DB(ctx).CreateTangent(ctx, tangentModel)
		require.NoError(t, err)

		// First create a session to get its ID
		httpReq, _ := http.NewRequest("POST", "/sessions", nil)
		req := `
			{
				"skillPath": "/valid-skillset/test-skill",
				"viewName": "valid-view",
				"sessionVariables": {
					"key1": "value1"
				},
				"inputArgs": {
					"input": "test input"
				}
			}`
		setRequestBodyAndHeader(t, httpReq, req)
		response := executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusCreated, response.Code)

		// Extract session ID from Location header
		location := response.Header().Get("Location")
		require.NotEmpty(t, location)
		//sessionID := location[strings.LastIndex(location, "/")+1:]

		// Create execution state with code verifier
		codeVerifier := "test_challenge"
		hashed := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hashed[:])

		// Create interactive session
		httpReq, _ = http.NewRequest("POST", "/sessions?interactive=true&code_challenge="+codeChallenge, nil)
		setRequestBodyAndHeader(t, httpReq, req)
		response = executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusOK, response.Code)

		var sessionResp session.InteractiveSessionRsp
		err = json.Unmarshal(response.Body.Bytes(), &sessionResp)
		assert.NoError(t, err)

		// Create execution state - this uses tangentAuthMiddleware
		httpReq, _ = http.NewRequest("POST", "/sessions/execution-state?code="+sessionResp.Code+"&code_verifier="+codeVerifier, nil)
		// Sign the request with tangent credentials
		err = signRequest(httpReq, privateKey, tangentID)
		require.NoError(t, err)
		response = executeTestRequest(t, httpReq, nil)
		require.Equal(t, http.StatusOK, response.Code)

		var tokenResp session.SessionTokenRsp
		err = json.Unmarshal(response.Body.Bytes(), &tokenResp)
		assert.NoError(t, err)

		// Get execution state to get the session ID
		httpReq, _ = http.NewRequest("GET", "/sessions/execution-state", nil)
		httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
		response = executeTestRequest(t, httpReq, nil)
		assert.Equal(t, http.StatusOK, response.Code)

		var executionState session.ExecutionState
		err = json.Unmarshal(response.Body.Bytes(), &executionState)
		assert.NoError(t, err)

		sessionID := executionState.SessionID.String()

		// Update execution state with audit log and verification key - this uses tangentAuthMiddleware
		httpReq, _ = http.NewRequest("PUT", "/sessions/execution-state", nil)
		httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
		httpReq.Header.Set("Content-Type", "application/json")
		updateReq := `
			{
				"sessionID": "` + sessionID + `",
				"statusSummary": "completed",
				"status": {
					"error": {
						"message": "test error",
						"code": "TEST_ERROR"
					},
					"auditLog": "dGVzdCBhdWRpdCBsb2c=",
					"auditLogVerificationKey": "dGVzdCBhdWRpdCBsb2cgdmVyaWZpY2F0aW9uIGtleQ=="
				}
			}`
		setRequestBodyAndHeader(t, httpReq, updateReq)
		// Sign the request with tangent credentials
		err = signRequest(httpReq, privateKey, tangentID)
		require.NoError(t, err)
		response = executeTestRequest(t, httpReq, nil)
		assert.Equal(t, http.StatusOK, response.Code)

		// Get audit log verification key
		httpReq, _ = http.NewRequest("GET", "/sessions/"+sessionID+"/auditlog/verification-key", nil)
		httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
		response = executeTestRequest(t, httpReq, nil)
		require.Equal(t, http.StatusOK, response.Code)

		var verificationKey session.AuditLogVerificationKey
		err = json.Unmarshal(response.Body.Bytes(), &verificationKey)
		assert.NoError(t, err)
		assert.Equal(t, []byte("test audit log verification key"), verificationKey.Key)

		// Test error cases
		t.Run("error cases", func(t *testing.T) {
			tests := []struct {
				name       string
				sessionID  string
				wantStatus int
			}{
				{
					name:       "invalid session ID",
					sessionID:  "invalid-uuid",
					wantStatus: http.StatusBadRequest,
				},
				{
					name:       "non-existent session",
					sessionID:  uuid.New().String(),
					wantStatus: http.StatusBadRequest,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					httpReq, _ := http.NewRequest("GET", "/sessions/"+tt.sessionID+"/auditlog/verification-key", nil)
					httpReq.Header.Set("Authorization", "Bearer "+tokenResp.Token)
					response := executeTestRequest(t, httpReq, nil)
					assert.Equal(t, tt.wantStatus, response.Code)
				})
			}
		})
	})
}
