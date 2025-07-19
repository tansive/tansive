// Package-level error variables for mcpservice, representing handler, request, and tool listing errors for the MCP service.
package mcpservice

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

var (
	// ErrMCPServiceError is the base error for MCP service errors.
	ErrMCPServiceError apperrors.Error = apperrors.New("mcp service error").SetStatusCode(http.StatusInternalServerError)

	// ErrMCPHandler is returned when the MCP handler is nil.
	ErrMCPHandler apperrors.Error = ErrMCPServiceError.New("mcp handler is nil")

	// ErrInvalidRequest is returned when the request is malformed or invalid.
	// Occurs when request body cannot be parsed or required fields are missing.
	ErrInvalidRequest apperrors.Error = ErrMCPServiceError.New("invalid request")

	// ErrListTools is returned when the list tools function is nil.
	ErrListTools apperrors.Error = ErrMCPServiceError.New("list tools function is nil")
)
