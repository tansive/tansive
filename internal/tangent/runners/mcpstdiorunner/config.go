package mcpstdiorunner

// Config defines the configuration for the MCP stdio runner, including command, arguments, environment, and version.
type Config struct {
	Version string            `json:"version"` // Version of the MCP client or protocol
	Command string            `json:"command"` // Command to launch the MCP server (e.g., "npx")
	Args    []string          `json:"args"`    // Arguments for the command
	Env     map[string]string `json:"env"`     // Environment variables for the MCP process
}
