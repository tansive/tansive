package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	log "github.com/rs/zerolog"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/common/httpx"
)

// UserAuthMiddleware handles authentication for both normal and single-user modes
func UserAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			log.Ctx(ctx).Warn().Msg("missing or invalid authorization header")
			httpx.ErrUnAuthorized("missing or invalid authorization header").Send(w)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		// First try normal token validation
		ctx, err := ValidateToken(ctx, token)
		if err == nil {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// If token validation failed and we're in single user mode, try that
		if config.IsTest() {
			ctx, err = handleSingleUserMode(ctx, token)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("authentication failed in single user mode")
				httpx.ErrUnAuthorized(err.Error()).Send(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// If we get here, token validation failed and we're not in single user mode
		log.Ctx(ctx).Warn().Err(err).Msg("token validation failed")
		httpx.ErrUnAuthorized("invalid authorization. login required").Send(w)
	})
}

// handleSingleUserMode processes authentication in single-user mode
func handleSingleUserMode(ctx context.Context, token string) (context.Context, error) {
	if token != config.Config().Auth.TestUserToken {
		return ctx, fmt.Errorf("invalid token in single user mode")
	}
	ctx, err := setDefaultSingleUserContext(ctx)
	if err != nil {
		return ctx, fmt.Errorf("failed to set user context")
	}
	return ctx, nil
}

// setDefaultSingleUserContext sets up the context for single-user mode
func setDefaultSingleUserContext(ctx context.Context) (context.Context, error) {
	ctx = catcommon.WithTenantID(ctx, catcommon.TenantId(config.Config().DefaultTenantID))

	catCtx := catcommon.GetCatalogContext(ctx)
	if catCtx == nil {
		catCtx = &catcommon.CatalogContext{}
	}

	catCtx.UserContext = &catcommon.UserContext{
		UserID: "default-user",
	}

	catCtx.Subject = catcommon.SubjectTypeUser

	ctx = catcommon.WithCatalogContext(ctx, catCtx)
	return ctx, nil
}
