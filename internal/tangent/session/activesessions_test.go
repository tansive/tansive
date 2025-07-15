package session

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/runners/stdiorunner"
	"github.com/tansive/tansive/internal/tangent/session/mcpservice"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/internal/tangent/test"
	"github.com/tansive/tansive/pkg/api"
)

func TestCreateSession(t *testing.T) {
	config.SetTestMode(true)
	ts := test.SetupTestCatalog(t)
	config.TestInit(t)
	SetTestMode(true)
	Init()
	stdiorunner.TestInit()
	token, expiresAt := test.AdoptView(t, ts.Catalog, "prod-view", ts.Token)
	serverContext := &ServerContext{
		SessionID:      uuid.New(),
		TenantID:       ts.TenantID,
		Catalog:        ts.Catalog,
		Variant:        "prod",
		SkillSet:       test.SkillsetPath(),
		Skill:          test.SkillsetAgent(),
		View:           "prod-view",
		ViewDefinition: test.GetViewDefinition("prod"),
	}
	ctx := context.Background()
	session, err := ActiveSessionManager().CreateSession(ctx, serverContext, token, expiresAt, tangentcommon.SessionTypeInteractive)
	require.NoError(t, err)
	err = session.fetchObjects(ctx)
	auditCtx, auditCancel := context.WithCancel(ctx)
	InitAuditLog(auditCtx, session)
	defer auditCancel()
	require.NoError(t, err)
	tCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	outWriter := tangentcommon.NewBufferedWriter()
	errWriter := tangentcommon.NewBufferedWriter()
	err = session.Run(tCtx, "", "k8s_troubleshooter", map[string]any{
		"prompt": "I'm getting a 500 error when I try to access the API",
	}, &tangentcommon.IOWriters{
		Out: outWriter,
		Err: errWriter,
	})
	require.NoError(t, err)
	t.Logf("outWriter: %s", outWriter.String())
	t.Logf("errWriter: %s", errWriter.String())
	CreateSkillService()
	// wait for few milliseconds to ensure the skill service is running
	time.Sleep(100 * time.Millisecond)
	serviceEndpoint, goerr := config.GetSocketPath()
	require.NoError(t, goerr)
	client, goerr := api.NewClient(serviceEndpoint)
	require.NoError(t, goerr)
	defer client.Close()

	// // empty invocationID should return an error
	_, goerr = client.InvokeSkill(ctx, session.GetSessionID(), "", "k8s_troubleshooter", map[string]any{
		"prompt": "I'm getting a 500 error when I try to access the API",
	})
	require.Error(t, goerr)

	// reusing invocationID should return a loop error
	var invocationID string
	for k := range session.invocationIDs {
		invocationID = k
		break
	}
	response, goerr := client.InvokeSkill(ctx, session.GetSessionID(), invocationID, "k8s_troubleshooter", map[string]any{
		"prompt": "I'm getting a 500 error when I try to access the API",
	})
	require.NoError(t, goerr)
	require.True(t, isError(response))

	// for testing, append a new invocationID to the session
	invocationID = uuid.New().String()
	session.invocationIDs[invocationID] = session.viewDef
	response, goerr = client.InvokeSkill(ctx, session.GetSessionID(), invocationID, "list_pods", map[string]any{
		"labelSelector": "app=my-app",
	})
	if !assert.NoError(t, goerr) {
		t.Logf("error: %v", goerr.Error())
		if response != nil {
			t.Logf("response: %v", response)
		}
		t.Fail()
	}
	if response != nil {
		t.Logf("response output: %s", response.Output["content"])
	}

	response, goerr = client.InvokeSkill(ctx, session.GetSessionID(), invocationID, "list_pods", map[string]any{
		"labelSelector": "app=my-app",
	})
	require.NoError(t, goerr)
	require.False(t, isError(response))
	t.Logf("response output: %v", response)

	response, goerr = client.InvokeSkill(ctx, session.GetSessionID(), invocationID, "restart_deployment", map[string]any{
		"deployment": "my-app",
	})
	require.NoError(t, goerr)
	require.True(t, isError(response))
	t.Logf("response outpu: %v", response)

	// // test get tools
	skills, goerr := client.GetSkills(ctx, session.GetSessionID())
	require.NoError(t, goerr)
	require.NotNil(t, skills)
	t.Logf("skills: %v", skills)
}

func isError(response *api.SkillResult) bool {
	_, ok := response.Output["error"]
	return ok
}

func TestCreateMCPProxySession(t *testing.T) {
	config.SetTestMode(true)
	ts := test.SetupTestCatalog(t)
	config.TestInit(t)
	SetTestMode(true)
	Init()
	stdiorunner.TestInit()
	token, expiresAt := test.AdoptView(t, ts.Catalog, "dev-view", ts.Token)
	serverContext := &ServerContext{
		SessionID:      uuid.New(),
		TenantID:       ts.TenantID,
		Catalog:        ts.Catalog,
		Variant:        "dev",
		SkillSet:       test.MCPSkillsetPath(),
		Skill:          test.MCPSkillsetAgent(),
		View:           "dev-view",
		ViewDefinition: test.GetViewDefinition("dev"),
	}
	ctx := context.Background()
	mcpService, err := mcpservice.CreateMCPService()
	CreateSkillService()
	require.NoError(t, err)
	session, err := ActiveSessionManager().CreateSession(ctx, serverContext, token, expiresAt, tangentcommon.SessionTypeMCPProxy)
	require.NoError(t, err)
	err = session.fetchObjects(ctx)
	auditCtx, auditCancel := context.WithCancel(ctx)
	InitAuditLog(auditCtx, session)
	defer auditCancel()
	require.NoError(t, err)
	url, token, err := session.RunMCPProxy(ctx, "", "supabase_mcp", map[string]any{})
	require.NoError(t, err)
	t.Logf("url: %s", url)
	t.Logf("token: %s", token)

	// extract the URI from the url without the domain
	uri := ""
	if idx := strings.Index(url, "/session/"); idx != -1 {
		uri = url[idx:]
	} else {
		t.Fatalf("could not extract /session/.../mcp from url: %s", url)
	}

	testListTools(t, mcpService, uri, token)
	testInvokeTool(t, mcpService, uri, token)

}

// JSON-RPC envelope struct
type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}
type jsonrpcResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      int              `json:"id"`
	Result  json.RawMessage  `json:"result"`
	Error   *json.RawMessage `json:"error,omitempty"`
}

func testListTools(t *testing.T, srv *mcpservice.MCPServer, uri string, token string) {
	// Use the client library to generate the request payload
	listReq := mcp.ListToolsRequest{
		PaginatedRequest: mcp.PaginatedRequest{
			Request: mcp.Request{
				Method: "tools/list",
			},
		},
	}
	jsonReq := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  listReq.Params,
	}
	listReqBytes, err := json.Marshal(jsonReq)
	if err != nil {
		t.Fatalf("Failed to marshal ListToolsRequest: %v", err)
	}

	req := httptest.NewRequest("POST", uri, bytes.NewReader(listReqBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var rpcResp jsonrpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		t.Fatalf("Failed to decode JSON-RPC response: %v\nBody: %s", err, string(body))
	}
	if rpcResp.Error != nil {
		t.Fatalf("JSON-RPC error: %s", string(*rpcResp.Error))
	}
	var listResp mcp.ListToolsResult
	if err := json.Unmarshal(rpcResp.Result, &listResp); err != nil {
		t.Fatalf("Failed to decode ListToolsResult: %v\nBody: %s", err, string(rpcResp.Result))
	}
	if len(listResp.Tools) == 0 {
		t.Fatalf("Expected at least one tool, got none")
	}
}

func testInvokeTool(t *testing.T, srv *mcpservice.MCPServer, uri string, token string) {
	// Directly call the 'list_tables' tool with argument {"schemas": ["public"]}
	callReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_tables",
			Arguments: map[string]interface{}{"schemas": []string{"public"}},
		},
	}

	callJsonReq := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params:  callReq.Params,
	}

	callReqBytes, err := json.Marshal(callJsonReq)
	if err != nil {
		t.Fatalf("Failed to marshal CallToolRequest: %v", err)
	}

	req2 := httptest.NewRequest("POST", uri, bytes.NewReader(callReqBytes))
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	srv.Router.ServeHTTP(w2, req2)

	resp2 := w2.Result()
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	var rpcResp2 jsonrpcResponse
	if err := json.Unmarshal(body2, &rpcResp2); err != nil {
		t.Fatalf("Failed to decode JSON-RPC response: %v\nBody: %s", err, string(body2))
	}
	if rpcResp2.Error != nil {
		t.Fatalf("JSON-RPC error: %s", string(*rpcResp2.Error))
	}

	// Instead of using mcp.CallToolResult, define a local struct for the test
	type TextContent struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type CallToolResultText struct {
		Content []TextContent `json:"content"`
		IsError bool          `json:"isError,omitempty"`
	}

	var callResp CallToolResultText
	if err := json.Unmarshal(rpcResp2.Result, &callResp); err != nil {
		t.Fatalf("Failed to decode CallToolResultText: %v\nBody: %s", err, string(rpcResp2.Result))
	}
	if len(callResp.Content) == 0 {
		t.Errorf("Expected non-empty Content in CallToolResultText, got none. Full response: %s", string(rpcResp2.Result))
	}
	require.Contains(t, callResp.Content[0].Text, "integration_tokens")

	// Make SQL query

	callReq.Params.Name = "execute_sql"
	callReq.Params.Arguments = map[string]interface{}{"query": "SELECT * FROM support_tickets;"}

	callJsonReq.Params = callReq.Params
	callReqBytes, err = json.Marshal(callJsonReq)
	if err != nil {
		t.Fatalf("Failed to marshal CallToolRequest: %v", err)
	}
	req2 = httptest.NewRequest("POST", uri, bytes.NewReader(callReqBytes))
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 = httptest.NewRecorder()
	srv.Router.ServeHTTP(w2, req2)

	resp2 = w2.Result()
	defer resp2.Body.Close()

	body2, _ = io.ReadAll(resp2.Body)
	var rpcResp3 jsonrpcResponse
	if err := json.Unmarshal(body2, &rpcResp3); err != nil {
		t.Fatalf("Failed to decode JSON-RPC response: %v\nBody: %s", err, string(body2))
	}
	if rpcResp3.Error != nil {
		t.Fatalf("JSON-RPC error: %s", string(*rpcResp3.Error))
	}

	if err := json.Unmarshal(rpcResp3.Result, &callResp); err != nil {
		t.Fatalf("Failed to decode CallToolResultText: %v\nBody: %s", err, string(rpcResp3.Result))
	}
	if len(callResp.Content) == 0 {
		t.Errorf("Expected non-empty Content in CallToolResultText, got none. Full response: %s", string(rpcResp3.Result))
	}
}
