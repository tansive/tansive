// Package tangentcommon provides common types and interfaces for the Tangent runtime.
// It defines shared data structures, I/O utilities, and interfaces used across
// multiple packages in the Tangent system for skill execution and session management.
package tangentcommon

import (
	"context"
	"io"

	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/pkg/api"
)

// IOWriters provides stdout and stderr writers for command output.
// Both Out and Err must implement io.Writer for proper I/O handling.
type IOWriters struct {
	Out io.Writer // stdout writer, must implement io.Writer
	Err io.Writer // stderr writer, must implement io.Writer
}

// RunParams defines parameters for skill execution.
// Contains session and invocation identifiers along with skill configuration.
type RunParams struct {
	SessionID    string         // unique session identifier
	InvocationID string         // unique invocation identifier
	SkillName    string         // name of the skill to execute
	InputArgs    map[string]any // input arguments for skill execution
}

// SkillManager defines the interface for skill execution management.
// Provides methods for skill listing, context management, and skill execution.
type SkillManager interface {
	// GetSkills retrieves available skills for a session as LLM tools.
	GetSkills(ctx context.Context, sessionID string) ([]api.LLMTool, apperrors.Error)

	// GetContext retrieves a context value for a session and invocation.
	GetContext(ctx context.Context, sessionID, invocationID, name string) (any, apperrors.Error)

	// Run executes a skill with the given parameters.
	Run(ctx context.Context, params *RunParams) (map[string]any, apperrors.Error)
}
