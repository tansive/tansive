// Package mcpservice provides HTTP server functionality for the MCP service.
// It exposes a single route for MCP session handling.
package mcpservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/middleware"
	"github.com/tansive/tansive/internal/tangent/config"
)

// ListToolsFunc defines a function type for listing available MCP tools.
type ListToolsFunc func(ctx context.Context) ([]mcp.Tool, error)

// CallToolFunc defines a function type for invoking an MCP tool.
type CallToolFunc func(ctx context.Context, tool mcp.Tool, params mcp.CallToolParams) (*mcp.CallToolResult, error)

// FilterToolsFunc defines a function type for filtering MCP tools.
type FilterToolsFunc func(ctx context.Context, tools []mcp.Tool) []mcp.Tool

// MCPHandler is a generic interface for MCP session handlers, providing tool listing, invocation, and filtering.
type MCPHandler interface {
	MCPListTools(ctx context.Context) ([]mcp.Tool, error)
	MCPCallTool(ctx context.Context, tool mcp.Tool, params mcp.CallToolParams) (*mcp.CallToolResult, error)
	MCPFilterTools(ctx context.Context, tools []mcp.Tool) []mcp.Tool
}

// MCPEndpoint represents a registered MCP session endpoint, associating an MCP server with a handler.
type MCPEndpoint struct {
	server  *server.MCPServer // Underlying MCP server instance
	handler MCPHandler        // Handler for tool operations
}

// MCPServer provides the HTTP server for the MCP service, managing session routing and handler registration.
type MCPServer struct {
	Router   *chi.Mux // HTTP router for request handling
	sessions sync.Map // Concurrent map of session randoms to handlers
}

var s *MCPServer

// CreateMCPService creates a new MCPServer instance.
func CreateMCPService() (*MCPServer, error) {
	if s != nil {
		return s, nil
	}
	s = &MCPServer{}
	s.Router = chi.NewRouter()
	s.mountHandlers()
	return s, nil
}

// mountHandlers sets up the single MCP route.
func (s *MCPServer) mountHandlers() {
	s.Router.Use(middleware.RequestLogger)
	s.Router.Use(middleware.PanicHandler)
	s.Router.Route("/session/mcp", func(r chi.Router) {
		r.Post("/", s.handleMCP)
	})
}

// handleMCP is a handler for the MCP endpoint.
func (s *MCPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	const bearerPrefix = "Bearer "
	const tokenPrefix = "tn_"
	if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Missing or invalid Authorization header"}`)
		return
	}
	sessionToken := authHeader[len(bearerPrefix):]
	if len(sessionToken) <= len(tokenPrefix) || sessionToken[:len(tokenPrefix)] != tokenPrefix {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Invalid session token"}`)
		return
	}
	tokenRandom := sessionToken[len(tokenPrefix):]
	sum := sha256.Sum256([]byte(tokenRandom))
	random := hex.EncodeToString(sum[:])
	if endpointVal, ok := s.sessions.Load(random); ok {
		log.Ctx(r.Context()).Info().Msg("handleMCP")
		handler, _ := endpointVal.(*MCPEndpoint)
		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "Invalid JSON"}`)
			return
		}
		resp := handler.server.HandleMessage(r.Context(), raw)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"error": "Session not found"}`)
}

// ListenAndServe starts the MCP server on port 8627.
func (s *MCPServer) ListenAndServe() error {
	addr := ":8627"
	fmt.Printf("MCPService listening on %s\n", addr)
	return http.ListenAndServe(addr, s.Router)
}

// NewMCPSession registers a new MCP session handler and returns the url, token, and random.
func NewMCPSession(ctx context.Context, handler MCPHandler) (string, string, string, apperrors.Error) {
	if s == nil {
		return "", "", "", ErrMCPServiceError.Msg("mcp service not initialized")
	}
	random := generateRandomString(32) // 128 bits = 32 hex chars
	token := "tn_" + random
	sum := sha256.Sum256([]byte(random))
	random = hex.EncodeToString(sum[:])
	if handler == nil {
		return "", "", "", ErrMCPHandler
	}
	srv := server.NewMCPServer(
		"tansive-mcp-server",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithToolFilter(handler.MCPFilterTools),
	)

	loadTools(ctx, srv, handler)

	endpoint := &MCPEndpoint{
		server:  srv,
		handler: handler,
	}

	s.sessions.Store(random, endpoint)
	url := fmt.Sprintf("http://" + config.Config().MCP.HostName + ":" + config.Config().MCP.Port + "/session/mcp")
	return url, token, random, nil
}

func StopMCPSession(ctx context.Context, random string) apperrors.Error {
	if s == nil {
		return ErrMCPServiceError.Msg("mcp service not initialized")
	}
	s.sessions.Delete(random)
	return nil
}

func loadTools(ctx context.Context, srv *server.MCPServer, handler MCPHandler) apperrors.Error {
	if handler == nil || srv == nil {
		log.Ctx(ctx).Error().Msg("listTools or srv is nil")
		return ErrInvalidRequest.Msg("listTools or srv is nil")
	}
	tools, err := handler.MCPListTools(ctx)
	if err != nil {
		return ErrListTools.MsgErr("failed to list tools", err)
	}
	for _, tool := range tools {
		srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Print the input when a tool call is made
			//inputBytes, _ := json.MarshalIndent(req, "", "  ")
			toolName := req.Params.Name
			log.Ctx(ctx).Info().Str("toolName", toolName).Msg("tool call")
			// Ensure the tool name is set correctly in the forwarded request
			forwardReq := req
			forwardReq.Params.Name = toolName
			return handler.MCPCallTool(ctx, tool, forwardReq.Params)
		})
	}
	log.Ctx(ctx).Info().Int("numTools", len(tools)).Msg("loaded tools")
	return nil
}

// generateRandomString generates a random alphanumeric string (hex encoded, lowercase) of the given length (in hex chars).
func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		panic("unable to generate random string")
	}
	return hex.EncodeToString(bytes)
}
