package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tansive/tansive/internal/catalogsrv/auth/userauth"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

// setCatalogContext creates and configures a new CatalogContext
func setCatalogContext(ctx context.Context, viewDef *policy.ViewDefinition, tokenObj *Token) (*catcommon.CatalogContext, apperrors.Error) {
	_ = ctx
	catalogContext := &catcommon.CatalogContext{
		Catalog:   viewDef.Scope.Catalog,
		Variant:   viewDef.Scope.Variant,
		Namespace: viewDef.Scope.Namespace,
		CatalogID: tokenObj.GetCatalogID(),
	}

	sub := tokenObj.GetSubject()
	if strings.HasPrefix(sub, "user/") {
		catalogContext.UserContext = &catcommon.UserContext{
			UserID: strings.TrimPrefix(sub, "user/"),
		}
		catalogContext.Subject = catcommon.SubjectTypeUser
	} else if strings.HasPrefix(sub, "session/") {
		catalogContext.Subject = catcommon.SubjectTypeSession
		sessionID, err := uuid.Parse(strings.TrimPrefix(sub, "session/"))
		if err != nil {
			return nil, ErrInvalidToken.Msg("invalid session ID")
		}
		catalogContext.SessionContext = &catcommon.SessionContext{
			SessionID: sessionID,
		}
	}

	return catalogContext, nil
}

// ValidateToken validates the provided token and sets up the appropriate context
func ValidateToken(ctx context.Context, token string) (context.Context, error) {
	if token == "" {
		return ctx, ErrInvalidToken.Msg("empty token. login required")
	}

	tokenType, jwtToken, err := ParseAndValidateToken(ctx, token)
	if err != nil {
		return ctx, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	switch tokenType {
	case catcommon.IdentityTokenType:
		return handleIdentityToken(ctx, jwtToken)
	case catcommon.AccessTokenType:
		return handleAccessToken(ctx, jwtToken)
	default:
		return ctx, ErrInvalidToken.Msg("invalid token. login required")
	}
}

func handleIdentityToken(ctx context.Context, jwtToken *jwt.Token) (context.Context, error) {
	if jwtToken == nil {
		return ctx, ErrInvalidToken.Msg("login expired")
	}

	tokenObj, err := userauth.ResolveIdentityToken(ctx, jwtToken)
	if err != nil {
		return ctx, ErrUnauthorized.Msg("invalid or expired login: " + err.Error())
	}

	tenantID := tokenObj.GetTenantID()
	if tenantID == "" {
		return ctx, ErrMissingTenantID
	}

	ctx = catcommon.WithTenantID(ctx, catcommon.TenantId(tenantID))

	sub := tokenObj.GetSubject()
	if strings.HasPrefix(sub, "user/") {
		catalogContext := &catcommon.CatalogContext{
			UserContext: &catcommon.UserContext{
				UserID: strings.TrimPrefix(sub, "user/"),
			},
			Subject: catcommon.SubjectTypeUser,
		}
		ctx = catcommon.WithCatalogContext(ctx, catalogContext)
	} else {
		return ctx, ErrInvalidToken.Msg("invalid subject")
	}

	return ctx, nil
}

func handleAccessToken(ctx context.Context, jwtToken *jwt.Token) (context.Context, error) {
	if jwtToken == nil {
		return ctx, ErrInvalidToken.Msg("invalid token")
	}

	tokenObj, err := ResolveAccessToken(ctx, jwtToken)
	if err != nil {
		return ctx, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	view := tokenObj.GetView()
	viewDef := policy.ViewDefinition{}
	if err := json.Unmarshal(view.Rules, &viewDef); err != nil {
		return ctx, fmt.Errorf("%w: %v", ErrInvalidViewRules, err)
	}

	tenantID := tokenObj.GetTenantID()
	if tenantID == "" {
		return ctx, ErrMissingTenantID
	}

	ctx = policy.WithViewDefinition(ctx, &viewDef)
	ctx = catcommon.WithTenantID(ctx, catcommon.TenantId(tenantID))

	catalogContext, err := setCatalogContext(ctx, &viewDef, tokenObj)
	if err != nil {
		return ctx, err
	}
	ctx = catcommon.WithCatalogContext(ctx, catalogContext)

	return ctx, nil
}
