// Package logtrace provides logging and tracing utilities for the application.
// It integrates with zerolog for structured logging and supports request tracing.
package logtrace

import (
	"context"
)

// RequestIdFromContext extracts the request ID from the context.
// Returns an empty string if the context is nil or if no request ID is found.
func RequestIdFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	r, ok := ctx.Value("requestId").(string)
	if !ok {
		return ""
	}
	return r
}

// IsTraceEnabled reports whether request tracing is enabled.
// Currently returns false as tracing is not yet implemented.
func IsTraceEnabled() bool {
	return false
}
