package userauth

import (
	"context"
	"net/http"
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/common/httpx"
)

type loginUserRsp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func LoginUser(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	if !config.Config().SingleUserMode {
		return nil, ErrLoginNotSupported.Msg("login is only supported in single user mode")
	}

	ctx, err := setDefaultSingleUserContext(ctx)
	if err != nil {
		return nil, err
	}

	token, tokenExpiry, err := CreateIdentityToken(ctx, getIdentityTokenClaims(ctx))
	if err != nil {
		return nil, err
	}

	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response: &loginUserRsp{
			Token:     token,
			ExpiresAt: tokenExpiry,
		},
	}, nil
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

func getIdentityTokenClaims(ctx context.Context) map[string]any {
	userContext := catcommon.GetUserContext(ctx)
	if userContext == nil || userContext.UserID == "" {
		return nil
	}

	return map[string]any{
		"token_use": catcommon.IdentityTokenType,
		"sub":       "user/" + userContext.UserID,
	}
}

var _ = getIdentityTokenClaims
