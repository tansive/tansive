package auth

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/httpx"
)

const (
	AuthHeaderPrefix = "Bearer "
	GenericAuthError = "authentication failed"
)

// ContextMiddleware handles authentication and context setup for incoming requests
func ContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Skip authentication for test contexts
		if catcommon.GetTestContext(ctx) {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Ctx(ctx).Debug().Msg("missing authorization header")
			httpx.ErrUnAuthorized(GenericAuthError).Send(w)
			return
		}

		if !strings.HasPrefix(authHeader, AuthHeaderPrefix) {
			log.Ctx(ctx).Debug().Msg("invalid authorization header format")
			httpx.ErrUnAuthorized(GenericAuthError).Send(w)
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, AuthHeaderPrefix))
		if token == "" {
			log.Ctx(ctx).Debug().Msg("empty token")
			httpx.ErrUnAuthorized(GenericAuthError).Send(w)
			return
		}

		var err error
		ctx, err = ValidateToken(ctx, token)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("token validation failed")
			httpx.ErrUnAuthorized(GenericAuthError).Send(w)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
