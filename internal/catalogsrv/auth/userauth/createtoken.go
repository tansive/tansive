package userauth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/auth/keymanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/common/uuid"
)

// Reserved JWT claims that cannot be overwritten
var reservedClaims = map[string]bool{
	"tenant_id": true,
	"iss":       true,
	"exp":       true,
	"iat":       true,
	"nbf":       true,
	"aud":       true,
	"jti":       true,
	"ver":       true,
}

// CreateIdentityToken creates a new JWT identity token
func CreateIdentityToken(ctx context.Context, additionalClaims map[string]any) (string, time.Time, error) {
	tokenDuration, goerr := config.Config().Auth.GetDefaultTokenValidity()
	if goerr != nil {
		log.Ctx(ctx).Error().Err(goerr).Msg("unable to parse token duration")
		return "", time.Time{}, ErrUnableToParseTokenDuration.MsgErr("unable to parse token duration", goerr)
	}

	tokenExpiry := time.Now().Add(tokenDuration)

	// Generate a unique token ID
	tokenID := uuid.New()

	claims := createTokenClaims(ctx, tokenID, tokenExpiry, additionalClaims)
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

// createTokenClaims creates the JWT claims for the identity token
func createTokenClaims(ctx context.Context, tokenID uuid.UUID, expiry time.Time, additionalClaims map[string]any) jwt.MapClaims {
	now := time.Now()
	claims := jwt.MapClaims{
		"token_use": catcommon.IdentityTokenType,
		"tenant_id": catcommon.GetTenantID(ctx),
		"iss":       config.Config().ServerHostName + ":" + config.Config().ServerPort,
		"exp":       jwt.NewNumericDate(expiry),
		"iat":       jwt.NewNumericDate(now),
		"nbf":       jwt.NewNumericDate(now.Add(-2 * time.Minute)), // 2-minute skew buffer
		"aud":       []string{"tansivesrv"},
		"jti":       tokenID.String(),
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
