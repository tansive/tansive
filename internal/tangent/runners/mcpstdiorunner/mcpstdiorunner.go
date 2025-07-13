package mcpstdiorunner

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/pkg/api"
)

type runner struct {
	config     Config
	client     *client.Client
	writers    []*tangentcommon.IOWriters
	clientLock sync.Mutex
}

func New(ctx context.Context, sessionID string, configMap map[string]any, writers ...*tangentcommon.IOWriters) (*runner, apperrors.Error) {
	// Parse config
	var config Config
	configData, err := json.Marshal(configMap)
	if err != nil {
		return nil, ErrInvalidConfig.MsgErr("failed to marshal config", err)
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, ErrInvalidConfig.MsgErr("failed to unmarshal config", err)
	}

	// Prepare env as []string
	env := make([]string, 0, len(config.Env))
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create MCP client
	mcpClient, err := client.NewStdioMCPClient(config.Command, env, config.Args...)
	if err != nil {
		return nil, ErrClientInit.MsgErr("failed to create MCP client", err)
	}

	// Initialize MCP client
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "tansive-mcp-client",
		Version: config.Version,
	}
	_, err = mcpClient.Initialize(ctx, initReq)
	if err != nil {
		mcpClient.Close()
		return nil, ErrClientInit.MsgErr("failed to initialize MCP client", err)
	}

	r := &runner{
		config:  config,
		client:  mcpClient,
		writers: writers,
	}
	return r, nil
}

func (r *runner) ID() string {
	return "system.mcp.stdio"
}

func (r *runner) AddWriters(writers ...*tangentcommon.IOWriters) {
	r.clientLock.Lock()
	defer r.clientLock.Unlock()
	if len(writers) == 0 {
		return
	}
	r.writers = append(r.writers, writers...)
}

// Runs Tansive skill. This is not supported for MCP stdio runner.
func (r *runner) Run(ctx context.Context, args *api.SkillInputArgs) apperrors.Error {
	return nil
}

func (r *runner) RunMCP(ctx context.Context, args *api.SkillInputArgs) (*mcp.CallToolResult, apperrors.Error) {
	if args == nil {
		return nil, apperrors.New("SkillInputArgs is nil")
	}
	// Call the tool on the MCP client
	r.clientLock.Lock()
	defer r.clientLock.Unlock()

	toolName := args.SkillName
	inputArgs := args.InputArgs

	callReq := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: inputArgs,
		},
	}
	result, err := r.client.CallTool(ctx, callReq)
	if err != nil {
		return nil, ErrToolCall.MsgErr("MCP tool call failed", err)
	}

	return result, nil
}

func (r *runner) FetchTools(ctx context.Context) ([]*api.LLMTool, apperrors.Error) {
	r.clientLock.Lock()
	defer r.clientLock.Unlock()
	toolsResult, err := r.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, ErrListTools.MsgErr("failed to list tools", err)
	}
	var tools []*api.LLMTool
	for _, t := range toolsResult.Tools {
		llmTool := &api.LLMTool{
			Name:        t.Name,
			Description: t.Description,
		}
		if b, err := json.Marshal(t.InputSchema); err == nil {
			llmTool.InputSchema = b
		}
		if b, err := json.Marshal(t.Annotations); err == nil {
			llmTool.Annotations = b
		}
		tools = append(tools, llmTool)
	}
	return tools, nil
}
