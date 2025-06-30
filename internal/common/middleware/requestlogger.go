// Package middleware provides HTTP middleware components for request logging, timeout handling,
// and panic recovery. It integrates with zerolog for structured logging and supports request
// tracing through unique request IDs.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/uuid"
)

// requestIdContextKey is a custom type for context key to store request IDs.
type requestIdContextKey string

const (
	requestIdKey    = requestIdContextKey("requestId")
	RequestIDHeader = "X-Tansive-Request-ID"
)

// RequestLogger creates middleware that logs incoming requests and adds a unique request ID
// to both the request context and response headers. It logs request details including URL,
// method, path, remote IP, and protocol. The request ID is used for request tracing.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()

		requestID := newRequestId()
		ctx = context.WithValue(ctx, requestIdKey, requestID)
		ctx = log.With().Str("request_id", requestID).Caller().Logger().WithContext(ctx)

		w.Header().Set(RequestIDHeader, requestID)

		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		requestURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)
		requestFields := map[string]any{
			"requestURL":    requestURL,
			"requestMethod": r.Method,
			"requestPath":   r.URL.Path,
			"remoteIP":      r.RemoteAddr,
			"proto":         r.Proto,
		}
		log.Ctx(ctx).Info().Fields(requestFields).Msg("incoming request")

		defer func() {
			log.Ctx(ctx).Info().
				Str("duration", fmt.Sprintf("%dms", time.Since(start).Milliseconds())).
				Msg("request completed")
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// newRequestId generates a unique request identifier. It attempts to create a UUID first,
// falling back to a timestamp-based ID if UUID generation fails.
func newRequestId() string {
	u, err := uuid.NewRandom()
	if err == nil {
		return u.String()
	}
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
}
