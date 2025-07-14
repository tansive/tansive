package mcpstdiorunner

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/pkg/api"
)

func TestMCPStdioRunner_FetchTools(t *testing.T) {
	if os.Getenv("SUPABASE_ACCESS_TOKEN") == "" {
		t.Skip("SUPABASE_ACCESS_TOKEN not set; skipping integration test")
	}
	if os.Getenv("SUPABASE_PROJECT") == "" {
		t.Skip("SUPABASE_PROJECT not set; skipping integration test")
	}
	/*
		command npx
		args [-y @supabase/mcp-server-supabase@latest --project-ref=xikujfynyknkfbopgewg]
		env [SUPABASE_ACCESS_TOKEN=sbp_33325c88de358fe86432893e00fc7f2831ba7073]
	*/
	/*
		command npx
		args [-y @supabase/mcp-server-supabase@latest --project-ref=xikujfynyknkfbopgewg]
		env [SUPABASE_ACCESS_TOKEN=sbp_33325c88de358fe86432893e00fc7f2831ba7073]
	*/
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	configMap := map[string]any{
		"version": "1.0.0",
		"command": "npx",
		"args": []any{
			"-y",
			"@supabase/mcp-server-supabase@latest",
			"--project-ref=" + os.Getenv("SUPABASE_PROJECT"),
		},
		"env": map[string]any{
			"SUPABASE_ACCESS_TOKEN": os.Getenv("SUPABASE_ACCESS_TOKEN"),
		},
	}

	var outBuf bytes.Buffer
	writers := &tangentcommon.IOWriters{Out: &outBuf}
	r, err := New(ctx, "test-session", configMap, writers)
	require.NoError(t, err)
	require.NotNil(t, r)

	tools, err2 := r.FetchTools(ctx)
	require.NoError(t, err2)
	require.NotEmpty(t, tools)

	var found bool
	for _, tool := range tools {
		t.Logf("Tool: %s - %s", tool.Name, tool.Description)
		if tool.Name == "list_tables" {
			found = true
		}
	}
	require.True(t, found, "list_tables tool not found in tool list")

	// Call the list_tables tool
	args := &api.SkillInputArgs{
		SkillName: "list_tables",
		InputArgs: map[string]any{
			"schemas": []string{"public"},
		},
	}
	result, err3 := r.RunMCP(ctx, args)
	require.NoError(t, err3)
	require.NotNil(t, result)
	require.False(t, result.IsError)
	require.NotNil(t, result.Content)
	require.Len(t, result.Content, 1)
	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	require.Equal(t, "text", textContent.Type)
	require.Contains(t, textContent.Text, "support_tickets")
}
