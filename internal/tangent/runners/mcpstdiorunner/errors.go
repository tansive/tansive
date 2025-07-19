package mcpstdiorunner

import "github.com/tansive/tansive/internal/common/apperrors"

// Package-level error variables for mcpstdiorunner, representing configuration, initialization, and tool invocation errors.
// All errors are derived from ErrMCPClientRunnerError.
var (
	// ErrMCPClientRunnerError is the base error for the package.
	ErrMCPClientRunnerError = apperrors.New("mcp client runner error")

	// ErrInvalidConfig is returned for invalid configurations.
	// Occurs when the JSON configuration cannot be unmarshaled into a Config.
	ErrInvalidConfig = ErrMCPClientRunnerError.New("invalid config")

	// ErrClientInit is returned when the MCP client fails to initialize.
	ErrClientInit = ErrMCPClientRunnerError.New("client initialization failed")

	// ErrToolCall is returned when a tool call fails.
	ErrToolCall = ErrMCPClientRunnerError.New("tool call failed")

	// ErrListTools is returned when listing tools fails.
	ErrListTools = ErrMCPClientRunnerError.New("list tools failed")

	// ErrInvalidWriters is returned for invalid I/O writers.
	ErrInvalidWriters = ErrMCPClientRunnerError.New("invalid writers")
)
