// Package middleware provides HTTP middleware components for request logging, timeout handling,
// and panic recovery. It integrates with zerolog for structured logging and supports request
// tracing through unique request IDs.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/httpx"
)

// SetTimeout creates middleware that enforces a timeout for request handling. If the request
// exceeds the specified duration, it returns a timeout error response. The timeout is added
// to response headers for debugging purposes. The middleware handles panics and ensures
// proper cleanup of resources.
func SetTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			rw := httpx.NewResponseWriter(w)
			r = r.WithContext(ctx)

			rw.Header().Set("X-Tansive-Timeout", timeout.String())

			// Create a channel to signal when the handler is done
			done := make(chan struct{})
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Ctx(ctx).Error().Msgf("panic in handler: %v", r)
					}
					close(done)
				}()
				next.ServeHTTP(rw, r)
			}()

			select {
			case <-done:
				// Handler completed successfully
				return
			case <-ctx.Done():
				// Only write error if headers haven't been sent yet
				if !rw.Written() {
					httpx.ErrRequestTimeout().Send(w)
				}
				log.Ctx(ctx).Error().Msgf("request timed out")
				return
			}
		})
	}
}
