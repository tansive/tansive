package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/auth/keymanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

// RequiredClaims is a list of claims that must be present in the token
var RequiredClaims = []string{
	"view_id",
	"tenant_id",
	"iss",
	"aud",
	"jti",
	"exp",
	"iat",
	"ver",
}

// Token represents a JWT token with its associated claims and validation methods
type Token struct {
	token  *jwt.Token
	claims jwt.MapClaims
	view   *models.View
}

func ParseAndValidateToken(ctx context.Context, tokenString string) (catcommon.TokenType, *jwt.Token, apperrors.Error) {
	signingKey, err := keymanager.GetKeyManager().GetActiveKey(ctx)
	if err != nil {
		return catcommon.UnknownTokenType, nil, err
	}

	var token *jwt.Token
	var parseErr error
	token, parseErr = jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return signingKey.PublicKey, nil
	}, jwt.WithValidMethods([]string{"EdDSA"}))

	if parseErr != nil {
		log.Ctx(ctx).Error().Err(parseErr).Msg("failed to parse token")
		return catcommon.UnknownTokenType, nil, ErrUnableToParseToken.Err(parseErr)
	}

	if !token.Valid {
		return catcommon.UnknownTokenType, nil, ErrUnableToParseToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return catcommon.UnknownTokenType, nil, ErrUnableToParseToken
	}

	tokenType, ok := claims["token_use"].(string)
	if !ok {
		return catcommon.UnknownTokenType, nil, ErrUnableToParseToken
	}

	switch tokenType {
	case string(catcommon.AccessTokenType):
		return catcommon.AccessTokenType, token, nil
	case string(catcommon.IdentityTokenType):
		return catcommon.IdentityTokenType, token, nil
	default:
		return catcommon.UnknownTokenType, nil, ErrUnableToParseToken
	}
}

// ResolveAccessToken parses and validates a JWT token string
func ResolveAccessToken(ctx context.Context, token *jwt.Token) (*Token, apperrors.Error) {
	if token == nil {
		return nil, ErrUnableToParseToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrUnableToParseToken
	}

	view_id, ok := claims["view_id"].(string)
	if !ok {
		return nil, ErrUnableToParseToken
	}

	viewID, parseUUIDErr := uuid.Parse(view_id)
	if parseUUIDErr != nil {
		return nil, ErrUnableToParseToken.Err(parseUUIDErr)
	}

	tenantID, ok := claims["tenant_id"].(string)
	if !ok {
		return nil, ErrUnableToParseToken
	}

	ctx = catcommon.WithTenantID(ctx, catcommon.TenantId(tenantID))

	view, err := db.DB(ctx).GetView(ctx, viewID)
	if err != nil {
		return nil, err
	}

	if view.TenantID != catcommon.TenantId(tenantID) {
		return nil, ErrInvalidToken.Msg(fmt.Sprintf("view tenant ID %s does not match token tenant ID %s", view.TenantID, tenantID))
	}

	tokenObj := &Token{
		token:  token,
		claims: claims,
		view:   view,
	}

	if err := tokenObj.Validate(ctx); err != nil {
		return nil, err
	}

	return tokenObj, nil
}

// Validate checks if the token is valid and not expired
func (t *Token) Validate(ctx context.Context) apperrors.Error {
	// Check all required claims are present
	for _, claim := range RequiredClaims {
		if _, ok := t.claims[claim]; !ok {
			log.Ctx(ctx).Debug().Str("claim", claim).Msg("missing required claim")
			return ErrInvalidToken.Msg(fmt.Sprintf("missing required claim: %s", claim))
		}
	}

	// Check version
	ver, ok := t.claims["ver"].(string)
	if !ok {
		log.Ctx(ctx).Debug().Msg("token missing or invalid ver claim")
		return ErrInvalidToken.Msg("missing or invalid ver claim")
	}
	if ver != string(catcommon.TokenVersionV0_1) {
		log.Ctx(ctx).Debug().Str("got", ver).Str("expected", string(catcommon.TokenVersionV0_1)).Msg("invalid token version")
		return ErrInvalidToken.Msg(fmt.Sprintf("invalid token version: got %s, expected %s", ver, catcommon.TokenVersionV0_1))
	}

	now := time.Now()

	// Check expiration with skew
	exp, ok := t.claims["exp"].(float64)
	if !ok {
		log.Ctx(ctx).Debug().Msg("token missing or invalid exp claim")
		return ErrInvalidToken.Msg("missing or invalid exp claim")
	}
	if now.After(time.Unix(int64(exp), 0).Add(config.Config().Auth.GetClockSkewOrDefault())) {
		log.Ctx(ctx).Debug().Msg("token expired")
		return ErrInvalidToken.Msg("token expired")
	}

	// Check not before with skew
	if nbf, ok := t.claims["nbf"]; ok {
		nbfFloat, ok := nbf.(float64)
		if !ok {
			log.Ctx(ctx).Debug().Type("type", nbf).Msg("invalid nbf claim type")
			return ErrInvalidToken.Msg("invalid nbf claim type")
		}
		if now.Before(time.Unix(int64(nbfFloat), 0).Add(-config.Config().Auth.GetClockSkewOrDefault())) {
			log.Ctx(ctx).Debug().Msg("token not yet valid")
			return ErrInvalidToken.Msg("token not yet valid")
		}
	}

	// Check if token is too old
	iat, ok := t.claims["iat"].(float64)
	if !ok {
		log.Ctx(ctx).Debug().Msg("token missing or invalid iat claim")
		return ErrInvalidToken.Msg("missing or invalid iat claim")
	}
	issuedAt := time.Unix(int64(iat), 0)
	if time.Since(issuedAt) > config.Config().Auth.GetMaxTokenAgeOrDefault() {
		log.Ctx(ctx).Debug().Msg("token too old")
		return ErrInvalidToken.Msg("token too old")
	}

	// Check issuer
	iss, ok := t.claims["iss"].(string)
	if !ok {
		log.Ctx(ctx).Debug().Msg("token missing or invalid iss claim")
		return ErrInvalidToken.Msg("missing or invalid iss claim")
	}
	expectedIssuer := config.Config().ServerHostName + ":" + config.Config().ServerPort
	if iss != expectedIssuer {
		log.Ctx(ctx).Debug().Str("got", iss).Str("expected", expectedIssuer).Msg("invalid issuer")
		return ErrInvalidToken.Msg(fmt.Sprintf("invalid issuer: got %s, expected %s", iss, expectedIssuer))
	}

	// Check audience
	aud, ok := t.claims["aud"]
	if !ok {
		log.Ctx(ctx).Debug().Msg("token missing aud claim")
		return ErrInvalidToken.Msg("missing aud claim")
	}
	expectedAudience := "tansivesrv"

	switch v := aud.(type) {
	case string:
		if v != expectedAudience {
			log.Ctx(ctx).Debug().Str("got", v).Str("expected", expectedAudience).Msg("invalid audience")
			return ErrInvalidToken.Msg(fmt.Sprintf("invalid audience: got %s, expected %s", v, expectedAudience))
		}
	case []any:
		found := false
		for _, a := range v {
			if s, ok := a.(string); ok && s == expectedAudience {
				found = true
				break
			}
		}
		if !found {
			log.Ctx(ctx).Debug().Interface("got", v).Str("expected", expectedAudience).Msg("invalid audience")
			return ErrInvalidToken.Msg(fmt.Sprintf("invalid audience: got %v, expected %s", v, expectedAudience))
		}
	default:
		log.Ctx(ctx).Debug().Type("type", v).Msg("invalid audience type")
		return ErrInvalidToken.Msg("invalid audience type")
	}

	// Check JWT ID
	jti, ok := t.claims["jti"].(string)
	if !ok {
		log.Ctx(ctx).Debug().Msg("token missing or invalid jti claim")
		return ErrInvalidToken.Msg("missing or invalid jti claim")
	}
	if revocationChecker.IsRevoked(jti) {
		log.Ctx(ctx).Debug().Str("jti", jti).Msg("token revoked")
		return ErrInvalidToken.Msg("token revoked")
	}

	return nil
}

// Get retrieves a claim value from the token
func (t *Token) Get(key string) (any, bool) {
	if t.claims == nil {
		return nil, false
	}
	val, ok := t.claims[key]
	return val, ok
}

// GetString retrieves a string claim value from the token
func (t *Token) GetString(key string) (string, bool) {
	val, ok := t.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

func (t *Token) GetTokenUse() catcommon.TokenType {
	tokenType, ok := t.Get("token_use")
	if !ok {
		return catcommon.UnknownTokenType
	}
	s, ok := tokenType.(string)
	if !ok {
		return catcommon.UnknownTokenType
	}
	return catcommon.TokenType(s)
}

func (t *Token) GetSubject() string {
	subject, ok := t.Get("sub")
	if !ok {
		return ""
	}
	s, ok := subject.(string)
	if !ok {
		return ""
	}
	return s
}

func (t *Token) GetTenantID() string {
	tenantID, ok := t.Get("tenant_id")
	if !ok {
		return ""
	}
	s, ok := tenantID.(string)
	if !ok {
		return ""
	}
	return s
}

func (t *Token) GetCatalogID() uuid.UUID {
	if t.view == nil {
		return uuid.Nil
	}
	return t.view.CatalogID
}

// GetUUID retrieves a UUID claim value from the token
func (t *Token) GetUUID(key string) (uuid.UUID, bool) {
	str, ok := t.GetString(key)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(str)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// GetViewID returns the view ID associated with the token
func (t *Token) GetViewID() uuid.UUID {
	if t.view == nil {
		return uuid.Nil
	}
	return t.view.ViewID
}

// GetExpiry returns the token's expiration time
func (t *Token) GetExpiry() time.Time {
	exp, ok := t.claims["exp"].(float64)
	if !ok {
		return time.Time{}
	}
	return time.Unix(int64(exp), 0)
}

// GetView returns the view associated with the token
func (t *Token) GetView() *models.View {
	return t.view
}

// GetRawToken returns the raw token string
func (t *Token) GetRawToken() string {
	if t.token == nil {
		return ""
	}
	return t.token.Raw
}
