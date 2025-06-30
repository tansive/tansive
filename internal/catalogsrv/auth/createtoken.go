package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/auth/keymanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

// TokenOptions contains options for token creation
type TokenOptions struct {
	ParentViewID      uuid.UUID
	ParentView        *policy.ViewDefinition
	CreateDerivedView bool
	AdditionalClaims  map[string]any
}

// TokenOption is a function that modifies TokenOptions
type TokenOption func(*TokenOptions)

// WithParentViewID sets the parent view ID
func WithParentViewID(id uuid.UUID) TokenOption {
	return func(o *TokenOptions) {
		o.ParentViewID = id
	}
}

// WithParentViewDefinition sets the parent view definition
func WithParentViewDefinition(view *policy.ViewDefinition) TokenOption {
	return func(o *TokenOptions) {
		o.ParentView = view
	}
}

// WithAdditionalClaims sets additional claims for the token
func WithAdditionalClaims(claims map[string]any) TokenOption {
	if claims == nil {
		claims = make(map[string]any)
	}
	return func(o *TokenOptions) {
		o.AdditionalClaims = claims
	}
}

// CreateDerivedView indicates that a derived view should be created
func CreateDerivedView() TokenOption {
	return func(o *TokenOptions) {
		o.CreateDerivedView = true
	}
}

// Reserved JWT claims that cannot be overwritten
var reservedClaims = map[string]bool{
	"view_id":   true,
	"tenant_id": true,
	"iss":       true,
	"exp":       true,
	"iat":       true,
	"nbf":       true,
	"aud":       true,
	"jti":       true,
	"ver":       true,
}

// CreateAccessToken creates a new JWT token for the given view
func CreateAccessToken(ctx context.Context, derivedView *models.View, opts ...TokenOption) (string, time.Time, apperrors.Error) {
	options := &TokenOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if derivedView == nil {
		return "", time.Time{}, ErrInvalidView.Msg("derived view is required")
	}

	if derivedView.Label == "" {
		return "", time.Time{}, ErrInvalidView.Msg("derived view label is required")
	}

	if options.CreateDerivedView {
		if err := db.DB(ctx).CreateView(ctx, derivedView); err != nil {
			return "", time.Time{}, err
		}
	}

	tokenDuration, goerr := config.Config().Auth.GetDefaultTokenValidity()
	if goerr != nil {
		log.Ctx(ctx).Error().Err(goerr).Msg("unable to parse token duration")
		return "", time.Time{}, ErrUnableToParseTokenDuration.MsgErr("unable to parse token duration", goerr)
	}

	tokenExpiry := time.Now().Add(tokenDuration)

	v := &models.ViewToken{
		ViewID:   derivedView.ViewID,
		ExpireAt: tokenExpiry,
	}
	if err := db.DB(ctx).CreateViewToken(ctx, v); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to create view token")
		return "", time.Time{}, ErrUnableToCreateView
	}

	claims := createTokenClaims(ctx, derivedView, v, tokenExpiry, options.AdditionalClaims)
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)

	signingKey, err := keymanager.GetKeyManager().GetActiveKey(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to get active signing key")
		return "", time.Time{}, err
	}

	token.Header["kid"] = signingKey.KeyID.String()

	tokenString, goerr := token.SignedString(signingKey.PrivateKey)
	if goerr != nil {
		log.Ctx(ctx).Error().Err(goerr).Msg("unable to sign token")
		return "", time.Time{}, ErrTokenGeneration.MsgErr("unable to sign token", goerr)
	}

	return tokenString, tokenExpiry, nil
}

// createTokenClaims creates the JWT claims for the token
func createTokenClaims(ctx context.Context, view *models.View, token *models.ViewToken, expiry time.Time, additionalClaims map[string]any) jwt.MapClaims {
	now := time.Now()
	claims := jwt.MapClaims{
		"token_use": catcommon.AccessTokenType,
		"view_id":   view.ViewID.String(),
		"tenant_id": catcommon.GetTenantID(ctx),
		"iss":       config.Config().ServerHostName + ":" + config.Config().ServerPort,
		"exp":       jwt.NewNumericDate(expiry),
		"iat":       jwt.NewNumericDate(now),
		"nbf":       jwt.NewNumericDate(now.Add(-2 * time.Minute)), // 2-minute skew buffer
		"aud":       []string{"tansivesrv"},
		"jti":       token.TokenID.String(),
		"ver":       string(catcommon.TokenVersionV0_1),
	}

	for k, v := range additionalClaims {
		if reservedClaims[k] {
			log.Ctx(ctx).Warn().Str("claim", k).Msg("attempt to override reserved claim ignored")
			continue
		}
		claims[k] = v
	}

	return claims
}

// ViewContextKey is the key used to store view information in the context
type ViewContextKey string

const viewContextKey ViewContextKey = "TansiveView"

// AddViewToContext adds a view definition to the context
func AddViewToContext(ctx context.Context, viewDefinition *policy.ViewDefinition) context.Context {
	return context.WithValue(ctx, viewContextKey, viewDefinition)
}

// GetViewFromContext retrieves a view definition from the context
func GetViewFromContext(ctx context.Context) *policy.ViewDefinition {
	v, ok := ctx.Value(viewContextKey).(*policy.ViewDefinition)
	if !ok {
		return nil
	}
	return v
}
