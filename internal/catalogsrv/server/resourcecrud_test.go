package server

import (
	"net/http"
	"strconv"
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
)

func TestResourceCrud(t *testing.T) {
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

	// Create a resource
	httpReq, _ = http.NewRequest("POST", "/resources", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "valid-resource",
				"catalog": "valid-catalog",
				"variant": "valid-variant",
				"namespace": "",
				"path": "/",
				"description": "This is a valid resource"
			},
			"spec": {
				"schema": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"value": {
							"type": "integer"
						}
					}
				},
				"value": {
					"name": "test-resource",
					"value": 42
				},
				"annotations": null
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusCreated, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
	assert.Contains(t, response.Header().Get("Location"), "/resources/valid-resource")

	// Get the resource
	httpReq, _ = http.NewRequest("GET", "/resources/definition/valid-resource", nil)
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

	// Update the resource
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "valid-resource",
				"catalog": "valid-catalog",
				"variant": "valid-variant",
				"namespace": "",
				"path": "/",
				"description": "This is an updated resource"
			},
			"spec": {
				"schema": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"value": {
							"type": "integer"
						}
					}
				},
				"value": {
					"name": "updated-resource",
					"value": 100
				},
				"annotations": null
			}
		}`
	httpReq, _ = http.NewRequest("PUT", "/resources/definition/valid-resource", nil)
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusOK, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Get the updated resource
	httpReq, _ = http.NewRequest("GET", "/resources/definition/valid-resource", nil)
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

	// Delete the resource
	httpReq, _ = http.NewRequest("DELETE", "/resources/definition/valid-resource", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusNoContent, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Try to get the deleted resource
	httpReq, _ = http.NewRequest("GET", "/resources/definition/valid-resource", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusNotFound, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}

	// Try to update non-existing resource
	httpReq, _ = http.NewRequest("PUT", "/resources/definition/not-existing-resource", nil)
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	if !assert.Equal(t, http.StatusNotFound, response.Code) {
		t.Logf("Response: %v", response.Body.String())
		t.FailNow()
	}
}

func TestResourceList(t *testing.T) {
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
				"description": "Catalog for resource list test"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	response := executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusCreated, response.Code)
	testContext.CatalogContext.Catalog = "list-catalog"

	// Create a variant
	httpReq, _ = http.NewRequest("POST", "/variants", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Variant",
			"metadata": {
				"name": "list-variant",
				"description": "Variant for resource list test"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusCreated, response.Code)
	testContext.CatalogContext.Variant = "list-variant"

	// Create resources
	resources := []struct {
		Name        string
		Description string
		Value       map[string]any
	}{
		{
			"resource1",
			"First test resource",
			map[string]any{
				"name":  "test1",
				"value": 1,
			},
		},
		{
			"resource2",
			"Second test resource",
			map[string]any{
				"name":  "test2",
				"value": 2,
			},
		},
		{
			"internal",
			"Internal resource",
			map[string]any{
				"name":  "internal",
				"value": 3,
			},
		},
	}

	for _, r := range resources {
		req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "` + r.Name + `",
				"catalog": "list-catalog",
				"variant": "list-variant",
				"description": "` + r.Description + `"
			},
			"spec": {
				"schema": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"value": {
							"type": "integer"
						}
					}
				},
				"value": {
					"name": "` + r.Value["name"].(string) + `",
					"value": ` + strconv.Itoa(r.Value["value"].(int)) + `
				}
			}
		}`
		httpReq, _ = http.NewRequest("POST", "/resources", nil)
		setRequestBodyAndHeader(t, httpReq, req)
		response = executeTestRequest(t, httpReq, nil, testContext)
		assert.Equal(t, http.StatusCreated, response.Code)
	}

	// List resources
	httpReq, _ = http.NewRequest("GET", "/resources?catalog=list-catalog&variant=list-variant", nil)
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	response = executeTestRequest(t, httpReq, nil, testContext)
	require.Equal(t, http.StatusOK, response.Code)

	var result = make(map[string]json.RawMessage)
	err = json.Unmarshal(response.Body.Bytes(), &result)
	assert.NoError(t, err)

	// All resources should be present
	assert.Len(t, result, 3)
}

func TestResourceValue(t *testing.T) {
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
				"name": "value-catalog",
				"description": "Catalog for resource value test"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	httpReq.Header.Set("Authorization", "Bearer "+config.Config().Auth.TestUserToken)
	response := executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusCreated, response.Code)
	testContext.CatalogContext.Catalog = "value-catalog"

	// Create a variant
	httpReq, _ = http.NewRequest("POST", "/variants", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Variant",
			"metadata": {
				"name": "value-variant",
				"description": "Variant for resource value test"
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusCreated, response.Code)
	testContext.CatalogContext.Variant = "value-variant"

	// Create a resource
	httpReq, _ = http.NewRequest("POST", "/resources", nil)
	req = `
		{
			"apiVersion": "0.1.0-alpha.1",
			"kind": "Resource",
			"metadata": {
				"name": "value-resource",
				"catalog": "value-catalog",
				"variant": "value-variant",
				"namespace": "",
				"path": "/",
				"description": "Resource for value test"
			},
			"spec": {
				"schema": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"value": {
							"type": "integer"
						}
					}
				},
				"value": {
					"name": "initial-value",
					"value": 42
				},
				"annotations": null
			}
		}`
	setRequestBodyAndHeader(t, httpReq, req)
	response = executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusCreated, response.Code)

	// Get the resource value
	httpReq, _ = http.NewRequest("GET", "/resources/value-resource", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusOK, response.Code)

	var valueResponse map[string]any
	err = json.Unmarshal(response.Body.Bytes(), &valueResponse)
	assert.NoError(t, err)
	assert.Equal(t, "initial-value", valueResponse["name"])
	assert.Equal(t, float64(42), valueResponse["value"])

	// Update the resource value
	updateValue := `
		{
			"name": "updated-value",
			"value": 100
		}`
	httpReq, _ = http.NewRequest("PUT", "/resources/value-resource", nil)
	setRequestBodyAndHeader(t, httpReq, updateValue)
	response = executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusOK, response.Code)

	// Get the updated resource value
	httpReq, _ = http.NewRequest("GET", "/resources/value-resource", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusOK, response.Code)

	err = json.Unmarshal(response.Body.Bytes(), &valueResponse)
	assert.NoError(t, err)
	assert.Equal(t, "updated-value", valueResponse["name"])
	assert.Equal(t, float64(100), valueResponse["value"])

	// Try to update with invalid value (should fail schema validation)
	invalidValue := `
		{
			"name": "invalid-value",
			"value": "not-a-number"
		}`
	httpReq, _ = http.NewRequest("PUT", "/resources/value-resource", nil)
	setRequestBodyAndHeader(t, httpReq, invalidValue)
	response = executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusBadRequest, response.Code)

	// Try to get non-existent resource value
	httpReq, _ = http.NewRequest("GET", "/resources/non-existent-resource", nil)
	response = executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusNotFound, response.Code)

	// Try to update non-existent resource value
	httpReq, _ = http.NewRequest("PUT", "/resources/non-existent-resource", nil)
	setRequestBodyAndHeader(t, httpReq, updateValue)
	response = executeTestRequest(t, httpReq, nil, testContext)
	assert.Equal(t, http.StatusNotFound, response.Code)
}
