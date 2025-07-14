// Package mcpstdiorunner provides an implementation of the Runner interface for executing MCP tools via a stdio-based client process.
// It manages client initialization, configuration, tool invocation, and output handling for integration with the Tansive platform.
package mcpstdiorunner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/pkg/api"
)

// runner manages the lifecycle and execution of MCP tools via stdio, including client initialization, configuration, and I/O writers.
type runner struct {
	config     Config                     // Configuration for the MCP runner
	client     *client.Client             // Underlying MCP stdio client
	writers    []*tangentcommon.IOWriters // Output writers for capturing tool output
	clientLock sync.Mutex                 // Mutex to protect client and writers
}

// New creates and initializes a new runner for executing MCP tools via stdio, using the provided configuration and writers.
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

	if config.Command == "" {
		return nil, ErrInvalidConfig.Msg("command is required")
	}

	if len(config.Args) > 0 {
		cleanArgs := make([]string, 0, len(config.Args))
		for _, arg := range config.Args {
			if arg == "" {
				continue
			}
			cleanArgs = append(cleanArgs, strings.TrimSpace(arg))
		}
		config.Args = cleanArgs
	}

	if len(env) > 0 {
		cleanEnv := make([]string, 0, len(env))
		for _, e := range env {
			if e == "" {
				continue
			}
			cleanEnv = append(cleanEnv, strings.TrimSpace(e))
		}
		env = cleanEnv
	}

	fmt.Println("command", config.Command)
	fmt.Println("args", config.Args)
	fmt.Println("env", env)

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

// ID returns the unique identifier for this runner implementation.
func (r *runner) ID() string {
	return "system.mcp.stdio"
}

// AddWriters appends additional IOWriters to the runner for capturing tool output.
func (r *runner) AddWriters(writers ...*tangentcommon.IOWriters) {
	r.clientLock.Lock()
	defer r.clientLock.Unlock()
	if len(writers) == 0 {
		return
	}
	r.writers = append(r.writers, writers...)
}

// Run is a no-op for the MCP stdio runner, as direct Tansive skill execution is not supported.
func (r *runner) Run(ctx context.Context, args *api.SkillInputArgs) apperrors.Error {
	return nil
}

// RunMCP invokes an MCP tool using the underlying stdio client, passing the provided skill input arguments.
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

// FetchTools retrieves the list of available MCP tools from the underlying client and converts them to LLMTool format.
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
