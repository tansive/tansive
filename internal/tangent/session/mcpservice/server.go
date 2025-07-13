// Package mcpservice provides HTTP server functionality for the MCP service.
// It exposes a single route for MCP session handling.
package mcpservice

import (
	"context"
	"crypto/rand"
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

type ListToolsFunc func(ctx context.Context) ([]mcp.Tool, error)
type CallToolFunc func(ctx context.Context, tool mcp.Tool, params mcp.CallToolParams) (*mcp.CallToolResult, error)
type FilterToolsFunc func(ctx context.Context, tools []mcp.Tool) []mcp.Tool

// MCPHandler is a generic interface for MCP session handlers.
type MCPHandler interface {
	MCPListTools(ctx context.Context) ([]mcp.Tool, error)
	MCPCallTool(ctx context.Context, tool mcp.Tool, params mcp.CallToolParams) (*mcp.CallToolResult, error)
	MCPFilterTools(ctx context.Context, tools []mcp.Tool) []mcp.Tool
}

type MCPEndpoint struct {
	server  *server.MCPServer
	handler MCPHandler
}

// MCPServer provides the HTTP server for the MCP service.
type MCPServer struct {
	Router   *chi.Mux // HTTP router for request handling
	sessions sync.Map // map[string]MCPHandler
}

var s *MCPServer

// CreateNewMCPServer creates a new MCPServer instance.
func CreateNewMCPServer() (*MCPServer, error) {
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
	s.Router.Route("/session/{sessionRandomURI}/mcp", func(r chi.Router) {
		r.Get("/", s.handleMCP)
	})
}

// handleMCP is a handler for the MCP endpoint.
func (s *MCPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	sessionRandomURI := chi.URLParam(r, "sessionRandomURI")
	if val, ok := s.sessions.Load(sessionRandomURI); ok {
		log.Ctx(r.Context()).Info().Str("sessionRandomURI", sessionRandomURI).Msg("handleMCP")
		// If found, call the handler's HandleMCP method
		handler, _ := val.(*MCPEndpoint)
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

// NewMCPSession registers a new MCP session handler and returns the random URI.
func NewMCPSession(ctx context.Context, handler MCPHandler) (string, string, apperrors.Error) {
	random := generateRandomString(64)
	if handler == nil {
		return "", "", ErrMCPHandler
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
	url := fmt.Sprintf("http://" + config.Config().ServerHostName + ":" + "8627" + "/session/" + random + "/mcp")
	return url, random, nil
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
			inputBytes, _ := json.MarshalIndent(req, "", "  ")
			toolName := req.Params.Name
			log.Ctx(ctx).Info().Str("toolName", toolName).Str("input", string(inputBytes)).Msg("tool call")
			// Ensure the tool name is set correctly in the forwarded request
			forwardReq := req
			forwardReq.Params.Name = toolName
			return handler.MCPCallTool(ctx, tool, forwardReq.Params)
		})
	}
	log.Ctx(ctx).Info().Int("numTools", len(tools)).Msg("loaded tools")
	return nil
}

// generateRandomString generates a 64-character random alphanumeric string (hex encoded, lowercase).
func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		panic("unable to generate random string")
	}
	return hex.EncodeToString(bytes)
}
