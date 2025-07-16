package mcpstdiorunner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// contextKey is a type for context keys in this package
// (avoid collisions)
type contextKey string

const canSeeHelloKey contextKey = "canSeeHello"

func main() {
	supabaseCmd := "npx"
	supabaseArgs := []string{"-y", "@supabase/mcp-server-supabase@latest", "--project-ref=xikujfynyknkfbopgewg"}
	supabaseEnv := []string{"SUPABASE_ACCESS_TOKEN=sbp_33325c88de358fe86432893e00fc7f2831ba7073"}

	clientInst, err := client.NewStdioMCPClient(supabaseCmd, supabaseEnv, supabaseArgs...)
	if err != nil {
		log.Fatalf("Failed to create Supabase MCP client: %v", err)
	}
	defer clientInst.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "supabase-mcp-proxy",
		Version: "1.0.0",
	}

	_, err = clientInst.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize Supabase MCP client: %v", err)
	}

	// On startup, fetch the tool list from the client
	toolsResult, err := clientInst.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("Failed to list tools from client: %v", err)
	}

	toolFilter := func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
		// No filtering yet, just return all registered tools
		return tools
	}

	s := server.NewMCPServer(
		"MCP Proxy",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithToolFilter(toolFilter),
	)

	// Register each tool from the client on the server
	for _, tool := range toolsResult.Tools {
		serverTool := mcp.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
			Annotations: tool.Annotations,
		}
		// Register a handler that forwards the call to the client
		toolName := tool.Name // capture for closure
		s.AddTool(serverTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Print the input when a tool call is made
			inputBytes, _ := json.MarshalIndent(req, "", "  ")
			fmt.Printf("Tool call to '%s' with input:\n%s\n", toolName, string(inputBytes))
			// Ensure the tool name is set correctly in the forwarded request
			forwardReq := req
			forwardReq.Params.Name = toolName
			return clientInst.CallTool(ctx, forwardReq)
		})
	}

	http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		fmt.Println("auth", auth)
		canSeeHello := false
		if len(auth) > 7 && auth[:7] == "Bearer " {
			auth = auth[7:]
		}
		if auth == "i-need-hello" {
			canSeeHello = true
		}
		ctx := context.WithValue(r.Context(), canSeeHelloKey, canSeeHello)

		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		resp := s.HandleMessage(ctx, raw)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	log.Println("Listening on :8080 (proxy server)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
