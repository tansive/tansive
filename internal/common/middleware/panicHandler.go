// Package middleware provides HTTP middleware components for request logging, timeout handling,
// and panic recovery. It integrates with zerolog for structured logging and supports request
// tracing through unique request IDs.
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/httpx"
)

// PanicHandler creates middleware that recovers from panics in HTTP handlers. When a panic
// occurs, it logs the panic details and stack trace, then returns a generic error response
// to the client. The middleware ensures that the application continues running even if
// individual requests panic.
func PanicHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := httpx.NewResponseWriter(w)
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()

				log.Ctx(r.Context()).Error().
					Str("panic", fmt.Sprintf("%v", err)).
					Str("stack_trace", string(stack)).
					Msg("panic occurred")

				if !rw.Written() {
					httpx.ErrApplicationError("unable to process request").Send(rw)
				}
			}
		}()
		next.ServeHTTP(rw, r)
	})
}
