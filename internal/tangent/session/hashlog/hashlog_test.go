package hashlog

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashLogWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.tlog")

	// Generate a new ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	writer, err := NewHashLogWriter(logPath, 3, privKey)
	require.NoError(t, err)
	defer writer.Close()

	// Add entries
	entries := []map[string]any{
		{"event": "start", "id": 1},
		{"event": "progress", "id": 2},
		{"event": "end", "id": 3},
		{"event": "summary", "id": 4},
	}

	for _, e := range entries {
		require.NoError(t, writer.AddEntry(e))
	}

	require.NoError(t, writer.Flush())

	// Read and verify entries
	file, err := os.Open(logPath)
	require.NoError(t, err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var prevHash string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		var entry HashedLogEntry
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &entry), "line %d unmarshal", lineNum)

		// Recompute hash
		expectedHashInput, _ := json.Marshal(struct {
			Payload  map[string]any `json:"payload"`
			PrevHash string         `json:"prevHash"`
		}{
			Payload:  entry.Payload,
			PrevHash: entry.PrevHash,
		})
		hash := sha256.Sum256(expectedHashInput)
		expectedHash := fmt.Sprintf("%x", hash[:])
		require.Equal(t, expectedHash, entry.Hash, "line %d hash mismatch", lineNum)

		// Verify signature
		signInput, _ := json.Marshal(struct {
			Payload  map[string]any `json:"payload"`
			PrevHash string         `json:"prevHash"`
			Hash     string         `json:"hash"`
		}{
			Payload:  entry.Payload,
			PrevHash: entry.PrevHash,
			Hash:     entry.Hash,
		})
		sig, err := base64.StdEncoding.DecodeString(entry.Signature)
		require.NoError(t, err, "line %d base64 decode", lineNum)
		require.True(t, ed25519.Verify(pubKey, signInput, sig), "line %d signature verify failed", lineNum)

		// Chain validation
		require.Equal(t, prevHash, entry.PrevHash, "line %d prevHash mismatch", lineNum)
		prevHash = entry.Hash
	}

	require.NoError(t, scanner.Err())
}

func TestHashLogWriterErrors(t *testing.T) {
	// Test invalid file path
	_, err := NewHashLogWriter("/nonexistent/path/test.tlog", 3, []byte("invalid-key"))
	require.Error(t, err)

	// Test invalid private key
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.tlog")
	_, err = NewHashLogWriter(logPath, 3, []byte("invalid-key"))
	require.Error(t, err)
}

func TestHashLogWriterBufferBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.tlog")

	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Test with flush interval of 1
	writer, err := NewHashLogWriter(logPath, 1, privKey)
	require.NoError(t, err)
	defer writer.Close()

	// Add single entry - should flush immediately
	err = writer.AddEntry(map[string]any{"test": "value"})
	require.NoError(t, err)

	// Verify file has content
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.NotEmpty(t, content)

	// Test with larger flush interval
	logPath2 := filepath.Join(tmpDir, "test2.tlog")
	writer2, err := NewHashLogWriter(logPath2, 5, privKey)
	require.NoError(t, err)
	defer writer2.Close()

	// Add 3 entries - should not flush yet
	for i := 0; i < 3; i++ {
		err = writer2.AddEntry(map[string]any{"count": i})
		require.NoError(t, err)
	}

	// Verify file is empty (not flushed)
	content, err = os.ReadFile(logPath2)
	require.NoError(t, err)
	require.Empty(t, content)

	// Add 2 more entries to trigger flush
	for i := 3; i < 5; i++ {
		err = writer2.AddEntry(map[string]any{"count": i})
		require.NoError(t, err)
	}

	// Verify file has content after flush
	content, err = os.ReadFile(logPath2)
	require.NoError(t, err)
	require.NotEmpty(t, content)
}

func TestHashLogWriterConcurrent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.tlog")

	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	writer, err := NewHashLogWriter(logPath, 10, privKey)
	require.NoError(t, err)
	defer writer.Close()

	// Launch multiple goroutines to write concurrently
	const numGoroutines = 10
	const entriesPerGoroutine = 5
	done := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < entriesPerGoroutine; j++ {
				err := writer.AddEntry(map[string]any{
					"goroutine": id,
					"entry":     j,
				})
				require.NoError(t, err)
			}
			done <- struct{}{}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Flush and verify
	require.NoError(t, writer.Flush())

	// Read and verify all entries were written
	file, err := os.Open(logPath)
	require.NoError(t, err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	entryCount := 0
	for scanner.Scan() {
		entryCount++
		var entry HashedLogEntry
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &entry))
		require.NotEmpty(t, entry.Payload)
		require.NotEmpty(t, entry.Hash)
		require.NotEmpty(t, entry.Signature)
	}

	require.Equal(t, numGoroutines*entriesPerGoroutine, entryCount)
}

func TestHashLogWriterEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.tlog")

	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	writer, err := NewHashLogWriter(logPath, 3, privKey)
	require.NoError(t, err)
	defer writer.Close()

	// Test empty payload
	err = writer.AddEntry(map[string]any{})
	require.NoError(t, err)

	// Test payload with special characters
	err = writer.AddEntry(map[string]any{
		"special": "!@#$%^&*()_+{}|:\"<>?",
		"unicode": "你好世界",
		"numbers": 123.456,
		"bool":    true,
		"null":    nil,
	})
	require.NoError(t, err)

	// Test multiple close calls
	require.NoError(t, writer.Close())
	require.NoError(t, writer.Close()) // Should not panic or error

	// Verify file has content
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.NotEmpty(t, content)
}

func TestHashLogWriterFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.tlog")

	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	writer, err := NewHashLogWriter(logPath, 3, privKey)
	require.NoError(t, err)
	defer writer.Close()

	// Add some entries
	err = writer.AddEntry(map[string]any{"test": "value"})
	require.NoError(t, err)
	require.NoError(t, writer.Flush())

	// Verify file permissions
	info, err := os.Stat(logPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0644), info.Mode().Perm())
}
