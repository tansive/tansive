package server

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	srvsession "github.com/tansive/tansive/internal/catalogsrv/session"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/runners/stdiorunner"
	"github.com/tansive/tansive/internal/tangent/session"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/internal/tangent/test"
)

func TestHandleInteractiveSession(t *testing.T) {
	config.SetTestMode(true)
	ts := test.SetupTestCatalog(t)
	config.TestInit(t)
	session.SetTestMode(true)
	session.Init()
	stdiorunner.TestInit()
	token := test.AdoptDefaultView(t, ts.Catalog)
	t.Logf("Token: %s", token)

	sessionReq := createInteractiveSession(t, token)
	sessionReq.SessionType = tangentcommon.SessionTypeInteractive

	httpReq, _ := http.NewRequest("POST", "/sessions", nil)
	setRequestBodyAndHeader(t, httpReq, sessionReq)
	response := executeTestRequest(t, httpReq, nil)
	require.Equal(t, http.StatusOK, response.Code)
	err := validateNDJSON(response.Body.Bytes())
	require.NoError(t, err)
}

func createInteractiveSession(t *testing.T, token string) *tangentcommon.SessionCreateRequest {
	codeVerifier := "test_challenge"
	hashed := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hashed[:])
	httpReq, _ := http.NewRequest("POST", "/sessions?interactive=true&code_challenge="+codeChallenge, nil)
	req := `
		{
			"skillPath": "/skillsets/kubernetes-demo/k8s_troubleshooter",
			"viewName": "dev-view",
			"sessionVariables": {
				"key1": "value1",
				"key2": 123,
				"key3": true
			},
			"inputArgs": {
				"prompt": "My order system is not working. Can you use the tools you have to fix it?"
			}
		}`
	test.SetRequestBodyAndHeader(t, httpReq, req)
	// add the token to the header
	httpReq.Header.Set("Authorization", "Bearer "+token)
	response := test.ExecuteTestRequest(t, httpReq, nil)
	require.Equal(t, http.StatusOK, response.Code)

	var sessionResp srvsession.InteractiveSessionRsp
	err := json.Unmarshal(response.Body.Bytes(), &sessionResp)
	require.NoError(t, err)
	require.NotEmpty(t, sessionResp.Code)
	require.NotEmpty(t, sessionResp.TangentURL)
	return &tangentcommon.SessionCreateRequest{
		Code:         sessionResp.Code,
		CodeVerifier: codeVerifier,
	}
}

// ValidateNDJSON checks whether each line in the input is a valid JSON object.
// Returns nil if valid, or an error describing the first invalid line.
func validateNDJSON(data []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Bytes()
		trimmed := bytes.TrimSpace(line)

		if len(trimmed) == 0 {
			lineNum++
			continue
		}

		fmt.Printf("line: %s\n", string(trimmed))

		var tmp any
		if err := json.Unmarshal(trimmed, &tmp); err != nil {
			return fmt.Errorf("invalid JSON at line %d: %w", lineNum, err)
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning input: %w", err)
	}
	fmt.Printf("lineNum: %d\n", lineNum)

	return nil
}
