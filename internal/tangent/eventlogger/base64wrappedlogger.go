package eventlogger

import (
	"encoding/base64"
	"io"

	"github.com/rs/zerolog"
)

// base64WrappedLogger wraps a zerolog.Logger to encode output as base64.
// Implements io.Writer interface for integration with systems requiring base64 encoding.
type base64WrappedLogger struct {
	Logger zerolog.Logger // underlying zerolog logger
}

// Write encodes the input bytes as base64 and writes to the underlying logger.
// Returns the number of bytes written and any error encountered during writing.
func (l *base64WrappedLogger) Write(p []byte) (n int, err error) {
	encoded := base64.StdEncoding.EncodeToString(p)
	l.Logger.Write([]byte(encoded))
	return len(p), nil
}

// NewBase64WrappedLogger creates a base64-encoding wrapper around a zerolog logger.
// Returns an io.Writer that encodes all output as base64 before logging.
func NewBase64WrappedLogger(logger zerolog.Logger) io.Writer {
	return &base64WrappedLogger{Logger: logger}
}
