package server

import (
	"net/http"
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
)

func TestViewCrud(t *testing.T) {
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
	// Create a New Request
	httpReq, _ := http.NewRequest("POST", "/catalogs", nil)
	req := `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Catalog",
			"metadata": {
				"name": "valid-catalog",
				"description": "This is a valid catalog"
			}
		} `
	setRequestBodyAndHeader(t, httpReq, req)
	// set bearer token in header
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	// Execute Request
	response := executeTestRequest(t, httpReq, nil, testContext)
	// Check the response code
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
	//	testContext.CatalogContext.Variant = "valid-variant"

	// Create a view
	httpReq, _ = http.NewRequest("POST", "/views", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "View",
			"metadata": {
				"name": "valid-view",
				"catalog": "valid-catalog",
				"description": "This is a valid view"
			},
			"spec": {
				"rules": [{
					"intent": "Allow",
					"actions": ["system.catalog.list"],
					"targets": ["res://variants/valid-variant"]
				}]
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	// Check Location in header
	assert.Contains(t, response.Header().Get("Location"), "/views/valid-view")

	// Get the view
	httpReq, _ = http.NewRequest("GET", "/views/valid-view", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusOK, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	checkHeader(t, response.Header())

	rspType := make(map[string]any)
	err = json.Unmarshal(response.Body.Bytes(), &rspType)
	assert.NoError(t, err)

	reqType := make(map[string]any)
	err = json.Unmarshal([]byte(req), &reqType)
	assert.NoError(t, err)
	assert.Equal(t, reqType, rspType)

	// Update the view
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "View",
			"metadata": {
				"name": "valid-view",
				"catalog": "valid-catalog",
				"description": "This is a new description"
			},
			"spec": {
				"rules": [{
					"intent": "Allow",
					"actions": ["system.catalog.list"],
					"targets": ["res://variants/valid-variant"]
				}]
			}
		}`
	httpReq, _ = http.NewRequest("PUT", "/views/valid-view", nil)
	setRequestBodyAndHeader(t, httpReq, req)

	response = executeTestRequest(t, httpReq, nil, testContext)
	// Check the response code
	if !assert.Equal(t, http.StatusOK, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Get the updated view
	httpReq, _ = http.NewRequest("GET", "/views/valid-view", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusOK, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	checkHeader(t, response.Header())

	rspType = make(map[string]any)
	err = json.Unmarshal(response.Body.Bytes(), &rspType)
	assert.NoError(t, err)

	reqType = make(map[string]any)
	err = json.Unmarshal([]byte(req), &reqType)
	assert.NoError(t, err)
	assert.Equal(t, reqType, rspType)

	// Delete the view
	httpReq, _ = http.NewRequest("DELETE", "/views/valid-view", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusNoContent, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Try to get the deleted view
	httpReq, _ = http.NewRequest("GET", "/views/valid-view", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusBadRequest, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Try to update not existing view
	httpReq, _ = http.NewRequest("PUT", "/views/not-existing-view", nil)
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusBadRequest, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
}

func TestViewList(t *testing.T) {
	ctx := newDb()
	t.Cleanup(func() {
		db.DB(ctx).Close(ctx)
	})

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("PABCDE")

	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	config.Config().DefaultProjectID = string(projectID)
	config.Config().DefaultTenantID = string(tenantID)

	err := db.DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = db.DB(ctx).DeleteTenant(ctx, tenantID)
	})
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
				"name": "list-catalog",
				"description": "Catalog for view list test"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	response := executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusCreated, response.Code)
	testContext.CatalogContext.Catalog = "list-catalog"

	// Create views (two normal, one internal)
	views := []struct {
		Name        string
		Description string
	}{
		{"view1", "First test view"},
		{"view2", "Second test view"},
		{"internal", "Internal view"},
	}
	for _, v := range views {
		req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "View",
			"metadata": {
				"name": "` + v.Name + `",
				"catalog": "list-catalog",
				"description": "` + v.Description + `"
			},
			"spec": {
				"rules": [{
					"intent": "Allow",
					"actions": ["system.catalog.list"],
					"targets": []
				}]
			}
		}`
		httpReq, _ = http.NewRequest("POST", "/views", nil)
		setRequestBodyAndHeader(t, httpReq, req)
		response = executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusCreated, response.Code)
	}

	// List views
	httpReq, _ = http.NewRequest("GET", "/views?catalog=list-catalog", nil)
	// set bearer token in header
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	response = executeTestRequest(t, httpReq, nil, testContext)
	require.Equal(t, http.StatusOK, response.Code)

	var result struct {
		Views []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"views"`
	}
	err = json.Unmarshal(response.Body.Bytes(), &result)
	assert.NoError(t, err)

	assert.Len(t, result.Views, 3)
}
