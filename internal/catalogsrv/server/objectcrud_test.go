package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tidwall/sjson"
)

func TestCreateTenantProject(t *testing.T) {
	ctx := newDb()
	t.Cleanup(func() {
		db.DB(ctx).Close(ctx)
	})

	tenantID := catcommon.TenantId("TANAND")
	projectID := catcommon.ProjectId("PANAND")

	// Create the tenant for testing
	err := db.DB(ctx).CreateTenant(ctx, tenantID)
	if err == nil {
		t.Logf("Tenant created successfully")
	} else {
		t.Logf("Tenant creation failed: %v", err)
	}

	// Set the tenant ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)

	// Create the project for testing
	err = db.DB(ctx).CreateProject(ctx, projectID)
	if err == nil {
		t.Logf("Project created successfully")
	} else {
		t.Logf("Project creation failed: %v", err)
	}
}

func TestCatalogCreate(t *testing.T) {

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
		TenantId:  tenantID,
		ProjectId: projectID,
	}

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
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	// Execute Request
	response := executeTestRequest(t, httpReq, nil, testContext)

	// Check the response code
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Check Location in header
	assert.Contains(t, response.Header().Get("Location"), "/catalogs/valid-catalog")

	// Test large request body
	httpReq, _ = http.NewRequest("POST", "/catalogs", nil)
	// Create a request body larger than the limit (1MB)
	largeBody := make([]byte, 2*1024*1024) // 2MB
	httpReq.Body = io.NopCloser(bytes.NewReader(largeBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)

	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
}

func TestGetUpdateDeleteCatalog(t *testing.T) {
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
		TenantId:  tenantID,
		ProjectId: projectID,
	}

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
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	// Execute Request
	response := executeTestRequest(t, httpReq, nil, testContext)
	// Check the response code
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	testContext.CatalogContext.Catalog = "valid-catalog"
	// Create a New Request to get the catalog
	httpReq, _ = http.NewRequest("GET", "/catalogs/valid-catalog", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)

	// Check the response code
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

	// Create a New Request to get a non-existing catalog
	httpReq, _ = http.NewRequest("GET", "/catalogs/validcatalog", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	t.Logf("Response: %v", response.Body.String())
	require.NotEqual(t, http.StatusOK, response.Code)

	// Update the catalog
	req = `
{
	"apiVersion": "0.1.0-alpha.1",
	"kind": "Catalog",
	"metadata": {
		"name": "valid-catalog",
		"description": "This is a new description"
	}
} `
	httpReq, _ = http.NewRequest("PUT", "/catalogs/valid-catalog", nil)
	setRequestBodyAndHeader(t, httpReq, req)

	response = executeTestRequest(t, httpReq, nil, testContext)
	// Check the response code
	if !assert.Equal(t, http.StatusOK, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Create a New Request to get the catalog
	httpReq, _ = http.NewRequest("GET", "/catalogs/valid-catalog", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)

	// Check the response code
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

	// Delete the catalog
	httpReq, _ = http.NewRequest("DELETE", "/catalogs/valid-catalog", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)

	// Check the response code
	if !assert.Equal(t, http.StatusNoContent, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Create a New Request to get the deleted catalog
	httpReq, _ = http.NewRequest("GET", "/catalogs/valid-catalog", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	require.NotEqual(t, http.StatusOK, response.Code)
}

func TestVariantCrud(t *testing.T) {
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
				"catalog": "valid-catalog",
				"description": "This is a valid variant"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	// Check Location in header
	assert.Contains(t, response.Header().Get("Location"), "/variants")
	loc := response.Header().Get("Location")

	// Get the variant
	httpReq, _ = http.NewRequest("GET", loc, nil)
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

	// Create a new variant on the /variants endpoint
	httpReq, _ = http.NewRequest("POST", "/variants", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Variant",
			"metadata": {
				"name": "valid-variant2",
				"catalog": "valid-catalog",
				"description": "This is a valid variant"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	// Check Location in header
	assert.Contains(t, response.Header().Get("Location"), "/variants/")
	loc = response.Header().Get("Location")
	// Get the variant
	httpReq, _ = http.NewRequest("GET", loc+"?c=valid-catalog", nil)
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

	// Create a new variant by updating the testcontext
	testContext.CatalogContext.Catalog = "invalid-catalog"
	req, _ = sjson.Set(req, "metadata.variant", "valid-variant-3")
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusBadRequest, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Update the variant
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Variant",
			"metadata": {
				"name": "valid-variant",
				"catalog": "valid-catalog",
				"description": "This is a new description"
			}
		}`
	testContext.CatalogContext.Catalog = "valid-catalog"
	httpReq, _ = http.NewRequest("PUT", "/variants/valid-variant", nil)
	setRequestBodyAndHeader(t, httpReq, req)

	response = executeTestRequest(t, httpReq, nil, testContext)
	// Check the response code
	if !assert.Equal(t, http.StatusOK, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Create a New Request to get the variant
	httpReq, _ = http.NewRequest("GET", "/variants/valid-variant", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)

	// Check the response code
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

	// Delete the variant
	httpReq, _ = http.NewRequest("DELETE", "/variants/valid-variant", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)

	// Check the response code
	if !assert.Equal(t, http.StatusNoContent, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Create a New Request to get the deleted variant
	httpReq, _ = http.NewRequest("GET", "/variants/valid-variant", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)

	// Check the response code
	if !assert.Equal(t, http.StatusBadRequest, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Get the variant
	httpReq, _ = http.NewRequest("GET", "/variants/valid-variant", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusBadRequest, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
}

func TestNamespaceCrud(t *testing.T) {
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
				"catalog": "valid-catalog",
				"description": "This is a valid variant"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Create a namespace
	httpReq, _ = http.NewRequest("POST", "/namespaces?c=valid-catalog&v=valid-variant", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Namespace",
			"metadata": {
				"name": "valid-namespace",
				"description": "This is a valid namespace"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	// Check Location in header
	assert.Equal(t, "/namespaces/valid-namespace", response.Header().Get("Location"))
	// Get the namespace
	httpReq, _ = http.NewRequest("GET", "/namespaces/valid-namespace?v=valid-variant&c=valid-catalog", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusOK, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
}
