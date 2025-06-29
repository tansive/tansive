package server

import (
	"net/http"
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
)

func TestGetVersion(t *testing.T) {
	newDb()
	// Create a New Request
	req, _ := http.NewRequest("GET", "/version", nil)
	testContext := TestContext{
		TenantId:  "tenant1",
		ProjectId: "project1",
		CatalogContext: catcommon.CatalogContext{
			Catalog: "test-catalog",
		},
	}
	// Execute Request
	response := executeTestRequest(t, req, nil, testContext)

	// Check the response code
	require.Equal(t, http.StatusOK, response.Code)

	// Check headers
	checkHeader(t, response.Result().Header)

	compareJson(t,
		&GetVersionRsp{
			ServerVersion: "Tansive Catalog Server: " + catcommon.ServerVersion,
			ApiVersion:    catcommon.ApiVersion,
		}, response.Body.String())
}

func TestGetReadiness(t *testing.T) {
	newDb()
	// Create a New Request
	req, _ := http.NewRequest("GET", "/ready", nil)
	testContext := TestContext{
		TenantId:  "tenant1",
		ProjectId: "project1",
	}
	// Execute Request
	response := executeTestRequest(t, req, nil, testContext)

	// Check the response code
	require.Equal(t, http.StatusOK, response.Code)

	// Check headers
	checkHeader(t, response.Result().Header)

	// Check response body
	compareJson(t, map[string]string{
		"status": "ready",
	}, response.Body.String())
}

func TestGetJWKS(t *testing.T) {
	config.TestInit()
	// Create a New Request
	req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
	testContext := TestContext{
		TenantId:  "tenant1",
		ProjectId: "project1",
	}
	// Execute Request
	response := executeTestRequest(t, req, nil, testContext)

	// Check the response code
	require.Equal(t, http.StatusOK, response.Code)

	// Check headers
	checkHeader(t, response.Result().Header)

	// Check response body structure
	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			Crv string `json:"crv"`
			X   string `json:"x"`
		} `json:"keys"`
	}
	err := json.Unmarshal(response.Body.Bytes(), &jwks)
	require.NoError(t, err)
	require.Len(t, jwks.Keys, 1)
	require.Equal(t, "OKP", jwks.Keys[0].Kty)
	require.Equal(t, "sig", jwks.Keys[0].Use)
	require.Equal(t, "EdDSA", jwks.Keys[0].Alg)
	require.Equal(t, "Ed25519", jwks.Keys[0].Crv)
	require.NotEmpty(t, jwks.Keys[0].X)
}
