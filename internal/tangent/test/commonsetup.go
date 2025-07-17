package test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"encoding/json"

	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
)

type TestSetup struct {
	Ctx       context.Context
	TenantID  catcommon.TenantId
	ProjectID catcommon.ProjectId
	Catalog   string
	Token     string
}

// SetupTestCatalog sets up a test catalog with default view and test objects
func SetupTestCatalog(t *testing.T) *TestSetup {
	ts := SetupTest(t)

	// Test successful adoption of default view
	token := AdoptDefaultView(t, "test-catalog")

	// Setup test objects with the adopted token
	SetupObjects(t, token)

	ts.Token = token

	return ts
}

// SetupTest initializes the test environment
func SetupTest(t *testing.T) *TestSetup {
	ctx := NewDb(t)
	t.Cleanup(func() {
		db.DB(ctx).Close(ctx)
	})

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("PABCDE")
	cfg := config.Config()
	cfg.DefaultTenantID = string(tenantID)
	cfg.DefaultProjectID = string(projectID)

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant for testing
	err := db.DB(ctx).CreateTenant(ctx, tenantID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.DB(ctx).DeleteTenant(ctx, tenantID)
	})

	// Create the project for testing
	err = db.DB(ctx).CreateProject(ctx, projectID)
	require.NoError(t, err)

	// Create a catalog
	httpReq, _ := http.NewRequest("POST", "/catalogs", nil)
	req := `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Catalog",
			"metadata": {
				"name": "test-catalog",
				"description": "Test catalog for adopt view"
			}
		}`
	SetRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	response := ExecuteTestRequest(t, httpReq, nil)
	require.Equal(t, http.StatusCreated, response.Code)

	return &TestSetup{
		Ctx:       ctx,
		TenantID:  tenantID,
		ProjectID: projectID,
		Catalog:   "test-catalog",
	}
}

// SetupObjects creates test objects in the catalog
func SetupObjects(t *testing.T, token string) {
	// Create a variant
	httpReq, _ := http.NewRequest("POST", "/variants", nil)
	req := `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Variant",
			"metadata": {
				"name": "dev",
				"description": "Development variant"
			}
		}`
	SetRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response := ExecuteTestRequest(t, httpReq, nil)
	require.Equal(t, http.StatusCreated, response.Code)

	// Create a prod variant
	httpReq, _ = http.NewRequest("POST", "/variants", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Variant",
			"metadata": {
				"name": "prod",
				"description": "Production variant"
			}
		}`
	SetRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response = ExecuteTestRequest(t, httpReq, nil)
	require.Equal(t, http.StatusCreated, response.Code)

	// Create a SkillSet for dev variant
	httpReq, _ = http.NewRequest("POST", "/skillsets?variant=dev", nil)
	req = string(SkillsetDef("dev"))
	SetRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response = ExecuteTestRequest(t, httpReq, nil)
	t.Logf("Skillset response: %s", response.Body.String())
	require.Equal(t, http.StatusCreated, response.Code)

	// Create a SkillSet for prod variant
	httpReq, _ = http.NewRequest("POST", "/skillsets?variant=prod", nil)
	req = string(SkillsetDef("prod"))
	SetRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response = ExecuteTestRequest(t, httpReq, nil)
	t.Logf("Skillset response: %s", response.Body.String())
	require.Equal(t, http.StatusCreated, response.Code)

	// Create a SkillSet for dev variant
	httpReq, _ = http.NewRequest("POST", "/skillsets?variant=dev", nil)
	req = string(getMCPSkillsetDef("dev"))
	SetRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response = ExecuteTestRequest(t, httpReq, nil)
	t.Logf("Skillset response: %s", response.Body.String())
	require.Equal(t, http.StatusCreated, response.Code)

	// Create a SkillSet for dev variant
	httpReq, _ = http.NewRequest("POST", "/skillsets?variant=prod", nil)
	req = string(getMCPSkillsetDef("prod"))
	SetRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response = ExecuteTestRequest(t, httpReq, nil)
	t.Logf("Skillset response: %s", response.Body.String())
	require.Equal(t, http.StatusCreated, response.Code)

	// Create dev view with all privileges
	httpReq, _ = http.NewRequest("POST", "/views", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "View",
			"metadata": {
				"name": "dev-view",
				"catalog": "test-catalog",
				"variant": "dev",
				"description": "View with full access to resources"
			},
			"spec": {
				"rules": [{
					"intent": "Allow",
					"actions": ["system.skillset.use","kubernetes.pods.list", "kubernetes.deployments.restart", "kubernetes.troubleshoot"],
					"targets": ["res://skillsets/skillsets/kubernetes-demo"]
				},
				{
					"intent": "Allow",
					"actions": ["system.skillset.use", "supabase.mcp.use", "supabase.tables.list", "supabase.sql.query"],
					"targets": ["res://skillsets/skillsets/supabase-demo"]
				}]
			}
		}`
	SetRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response = ExecuteTestRequest(t, httpReq, nil)
	require.Equal(t, http.StatusCreated, response.Code)

	// Create prod view with only non-sensitive actions
	httpReq, _ = http.NewRequest("POST", "/views", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "View",
			"metadata": {
				"name": "prod-view",
				"catalog": "test-catalog",
				"variant": "prod",
				"description": "View with only non-sensitive actions"
			},
			"spec": {
				"rules": [{
					"intent": "Allow",
					"actions": ["system.skillset.use","kubernetes.pods.list", "kubernetes.troubleshoot"],
					"targets": ["res://skillsets/skillsets/kubernetes-demo"]
				},
				{
					"intent": "Allow",
					"actions": ["system.skillset.use", "supabase.mcp.use", "supabase.tables.list", "supabase.sql.query"],
					"targets": ["res://skillsets/skillsets/supabase-demo"]
				}]
			}
		}`
	SetRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response = ExecuteTestRequest(t, httpReq, nil)
	require.Equal(t, http.StatusCreated, response.Code)
}

// AdoptDefaultView adopts the default view for a catalog
func AdoptDefaultView(t *testing.T, catalog string) string {
	httpReq, _ := http.NewRequest("POST", "/auth/default-view-adoptions/"+catalog, nil)
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	response := ExecuteTestRequest(t, httpReq, nil)
	require.Equal(t, http.StatusOK, response.Code)

	var adoptResponse struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	err := json.Unmarshal(response.Body.Bytes(), &adoptResponse)
	require.NoError(t, err)
	require.NotEmpty(t, adoptResponse.Token)
	require.NotEmpty(t, adoptResponse.ExpiresAt)

	return adoptResponse.Token
}

func AdoptView(t *testing.T, catalog, viewLabel, token string) (string, time.Time) {
	httpReq, _ := http.NewRequest("POST", "/auth/view-adoptions/"+catalog+"/"+viewLabel, nil)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response := ExecuteTestRequest(t, httpReq, nil)
	require.Equal(t, http.StatusOK, response.Code)

	var adoptResponse struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	err := json.Unmarshal(response.Body.Bytes(), &adoptResponse)
	require.NoError(t, err)
	require.NotEmpty(t, adoptResponse.Token)
	require.NotEmpty(t, adoptResponse.ExpiresAt)

	expiresAt, err := time.Parse(time.RFC3339, adoptResponse.ExpiresAt)
	require.NoError(t, err)

	return adoptResponse.Token, expiresAt
}
