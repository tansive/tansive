// Package runners provides the interface and factory for skill execution runners.
// It defines the Runner interface and provides factory methods to create appropriate
// runner instances based on skill configuration. The package supports multiple runner
// types including stdio-based execution for script and command running.
package runners

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/tangent/runners/mcpstdiorunner"
	"github.com/tansive/tansive/internal/tangent/runners/stdiorunner"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/pkg/api"
)

// Runner is the interface for all skill execution runners.
// Defines the contract for executing skills with input/output handling and lifecycle management.
type Runner interface {
	// ID returns the unique identifier for this runner instance.
	ID() string

	// AddWriters adds I/O writers for capturing command output.
	AddWriters(writers ...*tangentcommon.IOWriters)

	// Run executes a Tansive skill with the given arguments and context.
	Run(ctx context.Context, args *api.SkillInputArgs) apperrors.Error

	// Run MCP runs and returns an MCP style result
	RunMCP(ctx context.Context, args *api.SkillInputArgs) (*mcp.CallToolResult, apperrors.Error)

	// Fetch tools
	FetchTools(ctx context.Context) ([]*api.LLMTool, apperrors.Error)

	// Stop stops the runner
	Stop(ctx context.Context)
}

// NewRunner creates a new runner instance based on the runner definition.
// Returns the appropriate runner type and any error encountered during creation.
// Currently supports stdio runners for script and command execution.
func NewRunner(ctx context.Context, sessionID string, runnerDef catalogmanager.SkillSetSource, writers ...*tangentcommon.IOWriters) (Runner, apperrors.Error) {
	switch runnerDef.Runner {
	case catcommon.StdioRunnerID:
		return stdiorunner.New(ctx, sessionID, runnerDef.Config, writers...)
	case catcommon.MCPStdioRunnerID:
		return mcpstdiorunner.New(ctx, sessionID, runnerDef.Config, writers...)
	default:
		return nil, apperrors.New(fmt.Sprintf("invalid runner id: %s", runnerDef.Runner))
	}
}

// Init initializes the runners package and its dependencies.
// Must be called before using any runner functionality.
func Init() {
	stdiorunner.Init()
}
