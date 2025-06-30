// Package api provides client functionality for interacting with the Tansive Tangent service.
package api

import "encoding/json"

// LLMTool represents a skill or tool that can be invoked by the LLM.
// It contains metadata about the tool including its name, description, and input/output schemas.
type LLMTool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	InputSchema  json.RawMessage `json:"inputSchema,omitempty"`
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
}

// RunMode defines the execution mode for skill invocations.
// It determines how the skill should be executed and what behavior to expect.
type RunMode string

const (
	// RunModeInteractive indicates that the skill should run in interactive mode,
	// allowing for user interaction and real-time feedback during execution.
	RunModeInteractive RunMode = "interactive"

	// RunModeBatch indicates that the skill should run in batch mode,
	// executing without user interaction and returning results when complete.
	RunModeBatch RunMode = "batch"
)

// SkillInputArgs contains all the input parameters required for skill execution.
// It includes session information, invocation details, and the actual input arguments for the skill.
type SkillInputArgs struct {
	InvocationID     string         `json:"invocationID"`
	ServiceEndpoint  string         `json:"serviceEndpoint"`
	RunMode          RunMode        `json:"runMode"`
	SessionID        string         `json:"sessionID"`
	SkillName        string         `json:"skillName"`
	InputArgs        map[string]any `json:"inputArgs"`
	SessionVariables map[string]any `json:"sessionVariables"`
}

// TansiveSystemMessage is the standard system message that should be used
// when an operation is blocked by Tansive policy. It provides a consistent
// response format for policy violations and instructs users on how to proceed.
const TansiveSystemMessage = `
If an operation is blocked by Tansive policy (e.g., due to insufficient permissions or a denied capability), respond with the following message:
"Skill: <skill-name> is blocked by Tansive policy. Please contact the administrator of your Tansive system to obtain access."
Do not attempt to work around the restriction. Do not hallucinate or fabricate results. Respect the policy boundaries set by the system.`
