package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	uuidv7utils "github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/session/api"
)

func TestCreateSession(t *testing.T) {
	t.Skip("Skipping session tests")
	id := uuidv7utils.UUID7()
	reqBody := `
		{
			"id": "` + id.String() + `",
			"context": {
				"key1": "value1",
				"key2": "value2"
			}
		}
	`
	// Create a New Request
	req, _ := http.NewRequest("POST", "/sessions", nil)
	// Set the request body
	setRequestBodyAndHeader(t, req, reqBody)
	// Execute Request
	response := executeTestRequest(t, req, nil)

	// Check the response code
	require.Equal(t, http.StatusCreated, response.Code)

	// Check headers
	checkHeader(t, response.Result().Header)

	// Get the session ID from the response
	sessionID := response.Header().Get("Location")
	require.NotEmpty(t, sessionID, "Session ID should not be empty")

	// Get the session
	sessionReq, _ := http.NewRequest("GET", sessionID, nil)
	response = executeTestRequest(t, sessionReq, nil)
	require.Equal(t, http.StatusOK, response.Code, "Expected status code 200 OK for session retrieval")
	// Check headers
	checkHeader(t, response.Result().Header)
	// Compare the response body with the expected session data
	expectedSession := `
	{
		"id": "` + id.String() + `",
		"context": {
			"key1": "value1",
			"key2": "value2"
		}
	}`
	compareJson(t, expectedSession, response.Body.String())

	// create another session with the same ID
	req, _ = http.NewRequest("POST", "/sessions", nil)
	setRequestBodyAndHeader(t, req, reqBody)
	response = executeTestRequest(t, req, nil)
	require.Equal(t, http.StatusConflict, response.Code, "Expected status code 400 Bad Request for duplicate session creation")
	// Check headers
	checkHeader(t, response.Result().Header)

	// create new session with a different ID
	reqBody = `
		{
			"id": "` + uuidv7utils.UUID7().String() + `",
			"context": {
				"key1": "value1",
				"key2": "value2"
			}
		}`
	req, _ = http.NewRequest("POST", "/sessions", nil)
	setRequestBodyAndHeader(t, req, reqBody)
	response = executeTestRequest(t, req, nil)
	if !assert.Equal(t, http.StatusCreated, response.Code, "Expected status code 201 Created for new session") {
		t.Logf("Response body: %s", response.Body.String())
		t.FailNow()
	}

	// get all sessions
	req, _ = http.NewRequest("GET", "/sessions", nil)
	response = executeTestRequest(t, req, nil)
	require.Equal(t, http.StatusOK, response.Code, "Expected status code 200 OK for session listing")
	// Check headers
	checkHeader(t, response.Result().Header)
	// check the number of sessions
	sl := &api.ListSessionsResponse{}
	err := json.Unmarshal(response.Body.Bytes(), sl)
	require.NoError(t, err, "Failed to unmarshal sessions response")
	require.Equal(t, len(sl.Sessions), 2)

	// delete the first session
	req, _ = http.NewRequest("DELETE", sessionID, nil)
	response = executeTestRequest(t, req, nil)
	require.Equal(t, http.StatusNoContent, response.Code, "Expected status code 204 No Content for session deletion")
	// Check headers
	checkHeader(t, response.Result().Header)
	// try to get the deleted session
	sessionReq, _ = http.NewRequest("GET", sessionID, nil)
	response = executeTestRequest(t, sessionReq, nil)
	require.Equal(t, http.StatusBadRequest, response.Code, "Expected status code 404 Not Found for deleted session")
	// Check headers
	checkHeader(t, response.Result().Header)
	// try to delete the same session again
	req, _ = http.NewRequest("DELETE", sessionID, nil)
	response = executeTestRequest(t, req, nil)
	require.Equal(t, http.StatusBadRequest, response.Code, "Expected status code 404 Not Found for deleting non-existing session")
	// Check headers
	checkHeader(t, response.Result().Header)
	// try to get all sessions again
	req, _ = http.NewRequest("GET", "/sessions", nil)
	response = executeTestRequest(t, req, nil)
	require.Equal(t, http.StatusOK, response.Code, "Expected status code 200 OK for session listing after deletion")
	// Check headers
	checkHeader(t, response.Result().Header)
	// check the number of sessions again
	sl = &api.ListSessionsResponse{}
	err = json.Unmarshal(response.Body.Bytes(), sl)
	require.NoError(t, err, "Failed to unmarshal sessions response after deletion")
	require.Equal(t, len(sl.Sessions), 1, "Expected 1 session after deletion, got %d", len(sl.Sessions))
}
