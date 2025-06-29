package session

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/golang/snappy"
	"github.com/google/uuid"
	"github.com/tansive/tansive/internal/catalogsrv/config"
)

// isSnappyFramed checks for the standard Snappy framed stream header.
func isSnappyFramed(data []byte) bool {
	return len(data) >= 10 && bytes.HasPrefix(data, []byte{0xff, 0x06, 0x00, 0x00, 's', 'N', 'a', 'P', 'p', 'Y'})
}

// WriteAuditLogFile decodes a base64-encoded log and writes it with the appropriate extension.
func WriteAuditLogFile(ctx context.Context, sessionID uuid.UUID, auditLog string) (string, error) {
	auditLogBytes, err := base64.StdEncoding.DecodeString(auditLog)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 log: %w", err)
	}
	if len(auditLogBytes) == 0 {
		return "", errors.New("audit log is empty after base64 decode")
	}

	ext := ".tlog"
	if isSnappyFramed(auditLogBytes) {
		ext = ".ztlog"
	}

	logFilePath := filepath.Join(config.Config().AuditLog.GetPath(), sessionID.String()+ext)
	if err := os.WriteFile(logFilePath, auditLogBytes, 0600); err != nil {
		return "", fmt.Errorf("failed to write log file: %w", err)
	}
	return logFilePath, nil
}

// EncodeAuditLogFile reads a plain or compressed log file and returns it base64 encoded.
func EncodeAuditLogFile(ctx context.Context, sessionID uuid.UUID) (string, error) {
	basePath := filepath.Join(config.Config().AuditLog.GetPath(), sessionID.String())
	paths := []string{basePath + ".ztlog", basePath + ".tlog"}

	var logFilePath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			logFilePath = p
			break
		}
	}
	if logFilePath == "" {
		return "", fmt.Errorf("log file not found for session %s", sessionID)
	}

	f, err := os.Open(logFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	b64Encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if _, err := io.Copy(b64Encoder, f); err != nil {
		return "", fmt.Errorf("encoding failed: %w", err)
	}
	if err := b64Encoder.Close(); err != nil {
		return "", fmt.Errorf("base64 close failed: %w", err)
	}
	return buf.String(), nil
}

// CompressAndEncodeAuditLogFile compresses a log file with Snappy and base64-encodes the result.
func CompressAndEncodeAuditLogFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	b64Encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	snappyWriter := snappy.NewBufferedWriter(b64Encoder)

	if _, err := io.Copy(snappyWriter, f); err != nil {
		return "", fmt.Errorf("compression failed: %w", err)
	}
	if err := snappyWriter.Close(); err != nil {
		return "", fmt.Errorf("snappy close failed: %w", err)
	}
	if err := b64Encoder.Close(); err != nil {
		return "", fmt.Errorf("base64 close failed: %w", err)
	}
	return buf.String(), nil
}

// DecodeAndUncompressAuditLogFile decodes a base64-encoded log and uncompresses it with Snappy.
func DecodeAndUncompressAuditLogFile(encoded string, path string) error {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("base64 decode failed: %w", err)
	}

	snappyReader := snappy.NewReader(bytes.NewReader(decoded))

	// Write to a temporary file first
	tmpPath := path + ".tmp"
	outFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, snappyReader); err != nil {
		return fmt.Errorf("decompression failed: %w", err)
	}

	// Atomically move temp file to final path
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename to final path failed: %w", err)
	}

	return nil
}
