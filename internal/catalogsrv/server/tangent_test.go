package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/tangent"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestTangentCrud(t *testing.T) {
	ctx := newDb()
	t.Cleanup(func() {
		db.DB(ctx).Close(ctx)
	})

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("PABCDE")

	config.Config().DefaultProjectID = string(projectID)
	config.Config().DefaultTenantID = string(tenantID)
	config.Config().SingleUserMode = true

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

	// Test create tangent
	t.Run("create tangent", func(t *testing.T) {
		httpReq, _ := http.NewRequest("POST", "/tangents", nil)
		req := &tangent.TangentInfo{
			ID:                     uuid.New(),
			CreatedBy:              "test-user",
			URL:                    "http://test.tansive.dev:8468",
			Capabilities:           []catcommon.RunnerID{"system.commandrunner"},
			PublicKeyAccessKey:     []byte("test-access-key"),
			PublicKeyLogSigningKey: []byte("test-signing-key"),
			OnboardingKey:          config.Config().Tangent.OnboardingKey,
		}
		body, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}
		setRequestBodyAndHeader(t, httpReq, string(body))
		response := executeTestRequest(t, httpReq, nil)
		assert.Equal(t, http.StatusCreated, response.Code)

		// Extract tangent ID from Location header
		location := response.Header().Get("Location")
		require.NotEmpty(t, location)

		// Fetch and verify the created tangent
		createdTangent, err := db.DB(ctx).GetTangent(ctx, req.ID)
		require.NoError(t, err)
		assert.Equal(t, req.ID, createdTangent.ID)
		assert.Equal(t, string(tenantID), createdTangent.TenantID)

		// Unmarshal and verify Info
		var info tangent.TangentInfo
		err = json.Unmarshal(createdTangent.Info, &info)
		require.NoError(t, err)
		assert.Equal(t, req.CreatedBy, info.CreatedBy)
		assert.Equal(t, req.URL, info.URL)
		assert.Equal(t, req.Capabilities, info.Capabilities)

		// Verify PublicKey directly since it's stored as raw bytes
		assert.Equal(t, req.PublicKeyAccessKey, createdTangent.PublicKey)
	})

	// Test invalid tangent creation
	tests := []struct {
		name        string
		tangentSpec string
		wantStatus  int
	}{
		{
			name:        "missing body",
			tangentSpec: "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid json",
			tangentSpec: "{invalid json}",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name: "missing required fields",
			tangentSpec: `{
				"createdBy": "test-user",
				"url": "http://test.tansive.dev:8468",
				"capabilities": ["system.commandrunner"],
				"publicKeyAccessKey": "dGVzdC1hY2Nlc3Mta2V5",
				"publicKeyLogSigningKey": "dGVzdC1zaWduaW5nLWtleQ"
			}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpReq, _ := http.NewRequest("POST", "/tangents", nil)
			if tt.tangentSpec != "" {
				setRequestBodyAndHeader(t, httpReq, tt.tangentSpec)
			}
			response := executeTestRequest(t, httpReq, nil, testContext)
			assert.Equal(t, tt.wantStatus, response.Code)
		})
	}

	// Test multi-user mode restriction
	t.Run("multi-user mode restriction", func(t *testing.T) {
		config.Config().SingleUserMode = false
		httpReq, _ := http.NewRequest("POST", "/tangents", nil)
		req := `
			{
				"createdBy": "test-user",
				"url": "http://test.tansive.dev:8468",
				"capabilities": ["system.commandrunner"],
				"publicKeyAccessKey": "dGVzdC1hY2Nlc3Mta2V5",
				"publicKeyLogSigningKey": "dGVzdC1zaWduaW5nLWtleQ"
			}`
		setRequestBodyAndHeader(t, httpReq, req)
		response := executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})
}
