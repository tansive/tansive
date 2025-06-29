package hashlog

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderHashedLogToHTMLWithLargeLines(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.tlog")

	// Generate a new ed25519 key pair
	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	writer, err := NewHashLogWriter(logPath, 1, privKey)
	require.NoError(t, err)
	defer writer.Close()

	// Create a payload with a very large field that would exceed the default 64KB buffer
	largeData := make([]byte, 128*1024) // 128KB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Add an entry with large data
	err = writer.AddEntry(map[string]any{
		"event":      "test",
		"session_id": "test-session",
		"large_data": string(largeData),
	})
	require.NoError(t, err)

	// Add a few more normal entries
	for i := 0; i < 3; i++ {
		err = writer.AddEntry(map[string]any{
			"event":      "test",
			"session_id": "test-session",
			"count":      i,
		})
		require.NoError(t, err)
	}

	require.NoError(t, writer.Flush())

	// Test that RenderHashedLogToHTML can handle the large lines
	err = RenderHashedLogToHTML(logPath)
	require.NoError(t, err)

	// Verify that the HTML file was created
	htmlPath := strings.TrimSuffix(logPath, filepath.Ext(logPath)) + ".html"
	_, err = os.Stat(htmlPath)
	require.NoError(t, err)
}
