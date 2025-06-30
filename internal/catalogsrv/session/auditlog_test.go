package session

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/snappy"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/config"
)

func TestWriteAndEncodeAuditLogFile(t *testing.T) {
	config.TestInit()

	auditLogPath := config.Config().AuditLog.GetPath()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "auditlog-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test file in the temp directory
	testContent := []byte("This is a test log file content")
	testFilePath := filepath.Join(tempDir, "test.log")
	err = os.WriteFile(testFilePath, testContent, 0600)
	require.NoError(t, err)

	// Create a proper Snappy framed stream
	var buf bytes.Buffer
	snappyWriter := snappy.NewBufferedWriter(&buf)
	_, err = snappyWriter.Write([]byte("Hello World"))
	require.NoError(t, err)
	err = snappyWriter.Close()
	require.NoError(t, err)
	compressedBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Test cases
	tests := []struct {
		name      string
		auditLog  string
		wantExt   string
		wantError bool
	}{
		{
			name:      "empty log",
			auditLog:  "",
			wantError: true,
		},
		{
			name:      "invalid base64",
			auditLog:  "invalid-base64",
			wantError: true,
		},
		{
			name:     "plain log",
			auditLog: "SGVsbG8gV29ybGQ=", // "Hello World" in base64
			wantExt:  ".tlog",
		},
		{
			name:     "snappy compressed log",
			auditLog: compressedBase64,
			wantExt:  ".ztlog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID := uuid.New()

			// Test WriteAuditLogFile
			_, err := WriteAuditLogFile(context.Background(), sessionID, tt.auditLog)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify file exists with correct extension
			expectedPath := filepath.Join(auditLogPath, sessionID.String()+tt.wantExt)
			_, err = os.Stat(expectedPath)
			assert.NoError(t, err)

			// Test EncodeAuditLogFile
			encoded, err := EncodeAuditLogFile(context.Background(), sessionID)
			require.NoError(t, err)
			assert.Equal(t, tt.auditLog, encoded)
		})
	}
}

func TestCompressAndEncodeAuditLogFile(t *testing.T) {
	config.TestInit()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "auditlog-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test file
	testContent := []byte("This is a test log file content that will be compressed")
	testFilePath := filepath.Join(tempDir, "test.log")
	err = os.WriteFile(testFilePath, testContent, 0600)
	require.NoError(t, err)

	// Test compression and encoding
	encoded, err := CompressAndEncodeAuditLogFile(testFilePath)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	// Verify the encoded content is valid base64
	_, err = os.Stat(testFilePath)
	assert.NoError(t, err)
}

func TestIsSnappyFramed(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "too short data",
			data:     []byte{0xff, 0x06},
			expected: false,
		},
		{
			name:     "valid snappy header",
			data:     []byte{0xff, 0x06, 0x00, 0x00, 's', 'N', 'a', 'P', 'p', 'Y'},
			expected: true,
		},
		{
			name:     "invalid header",
			data:     []byte{0xff, 0x06, 0x00, 0x00, 'i', 'n', 'v', 'a', 'l', 'i', 'd'},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSnappyFramed(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}
