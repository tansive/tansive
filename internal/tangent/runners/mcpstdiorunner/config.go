package mcpstdiorunner

type Config struct {
	Version string            `json:"version"`
	Command string            `json:"command"` // e.g. "npx"
	Args    []string          `json:"args"`    // e.g. ["-y", "@supabase/mcp-server-supabase@latest", ...]
	Env     map[string]string `json:"env"`     // e.g. {"SUPABASE_ACCESS_TOKEN": "..."}
}
