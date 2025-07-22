package userauth

import (
	"context"
	"net/http"
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/common/httpx"
	"golang.org/x/crypto/bcrypt"
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

	password := r.URL.Query().Get("password")
	if password == "" {
		return nil, ErrLoginNotSupported.Msg("password is required")
	}
	// get the password hash from the runtime config
	userPassword := config.Config().SingleUserPasswordHash
	if userPassword == "" {
		err := onboardSingleUser(ctx, password)
		if err != nil {
			return nil, err
		}
		userPassword = config.Config().SingleUserPasswordHash
	}

	// compare the hashed password with the user password
	if err := bcrypt.CompareHashAndPassword([]byte(userPassword), []byte(password)); err != nil {
		return nil, ErrLoginNotSupported.Msg("invalid password")
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

func onboardSingleUser(ctx context.Context, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = config.SetSingleUserPassword(string(hashedPassword))
	if err != nil {
		return err
	}
	ctx = catcommon.WithTenantID(ctx, catcommon.TenantId(config.Config().DefaultTenantID))
	// delete all catalogs
	err = catalogmanager.DeleteAllCatalogs(ctx)
	if err != nil {
		return err
	}
	return nil
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
