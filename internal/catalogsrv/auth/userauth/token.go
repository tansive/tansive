package userauth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

// RevocationChecker defines the interface for checking if a token has been revoked
type RevocationChecker interface {
	IsRevoked(jti string) bool
}

// defaultRevocationChecker is a simple implementation that always returns false
type defaultRevocationChecker struct{}

func (c *defaultRevocationChecker) IsRevoked(jti string) bool {
	return false
}

var revocationChecker RevocationChecker = &defaultRevocationChecker{}

// SetRevocationChecker sets the revocation checker implementation
func SetRevocationChecker(checker RevocationChecker) {
	if checker == nil {
		checker = &defaultRevocationChecker{}
	}
	revocationChecker = checker
}

// RequiredClaims is a list of claims that must be present in the identity token
var RequiredClaims = []string{
	"tenant_id",
	"iss",
	"aud",
	"jti",
	"exp",
	"iat",
	"ver",
}

// IdentityToken represents a JWT identity token with its associated claims and validation methods
type IdentityToken struct {
	token  *jwt.Token
	claims jwt.MapClaims
}

// ResolveIdentityToken parses and validates a JWT identity token string
func ResolveIdentityToken(ctx context.Context, token *jwt.Token) (*IdentityToken, apperrors.Error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrUnableToParseToken
	}

	tenantID, ok := claims["tenant_id"].(string)
	if !ok {
		return nil, ErrUnableToParseToken
	}

	ctx = catcommon.WithTenantID(ctx, catcommon.TenantId(tenantID))

	tokenObj := &IdentityToken{
		token:  token,
		claims: claims,
	}

	if err := tokenObj.Validate(ctx); err != nil {
		return nil, err
	}

	return tokenObj, nil
}

// Validate checks if the identity token is valid and not expired
func (t *IdentityToken) Validate(ctx context.Context) apperrors.Error {
	// Check all required claims are present
	if err := t.validateRequiredClaims(ctx); err != nil {
		return err
	}

	// Check version
	if err := t.validateVersion(ctx); err != nil {
		return err
	}

	// Check timing claims
	if err := t.validateTimingClaims(ctx); err != nil {
		return err
	}

	// Check issuer
	if err := t.validateIssuer(ctx); err != nil {
		return err
	}

	// Check audience
	if err := t.validateAudience(ctx); err != nil {
		return err
	}

	// Check JWT ID
	if err := t.validateJWTID(ctx); err != nil {
		return err
	}

	return nil
}

// validateRequiredClaims checks that all required claims are present
func (t *IdentityToken) validateRequiredClaims(ctx context.Context) apperrors.Error {
	for _, claim := range RequiredClaims {
		if _, ok := t.claims[claim]; !ok {
			log.Ctx(ctx).Debug().Str("claim", claim).Msg("missing required claim")
			return ErrInvalidToken.Msg(fmt.Sprintf("missing required claim: %s", claim))
		}
	}
	return nil
}

// validateVersion checks the token version
func (t *IdentityToken) validateVersion(ctx context.Context) apperrors.Error {
	ver, ok := t.claims["ver"].(string)
	if !ok {
		log.Ctx(ctx).Debug().Msg("token missing or invalid ver claim")
		return ErrInvalidToken.Msg("missing or invalid ver claim")
	}
	if ver != string(catcommon.TokenVersionV0_1) {
		log.Ctx(ctx).Debug().Str("got", ver).Str("expected", string(catcommon.TokenVersionV0_1)).Msg("invalid token version")
		return ErrInvalidToken.Msg(fmt.Sprintf("invalid token version: got %s, expected %s", ver, catcommon.TokenVersionV0_1))
	}
	return nil
}

// validateTimingClaims checks expiration, not-before, and issued-at claims
func (t *IdentityToken) validateTimingClaims(ctx context.Context) apperrors.Error {
	now := time.Now()

	// Check expiration with skew
	if err := t.validateExpiration(ctx, now); err != nil {
		return err
	}

	// Check not before with skew
	if err := t.validateNotBefore(ctx, now); err != nil {
		return err
	}

	// Check if token is too old
	if err := t.validateIssuedAt(ctx); err != nil {
		return err
	}

	return nil
}

// validateExpiration checks the exp claim
func (t *IdentityToken) validateExpiration(ctx context.Context, now time.Time) apperrors.Error {
	exp, ok := t.claims["exp"].(float64)
	if !ok {
		log.Ctx(ctx).Debug().Msg("token missing or invalid exp claim")
		return ErrInvalidToken.Msg("missing or invalid exp claim")
	}
	if now.After(time.Unix(int64(exp), 0).Add(config.Config().Auth.GetClockSkewOrDefault())) {
		log.Ctx(ctx).Debug().Msg("token expired")
		return ErrInvalidToken.Msg("login expired")
	}
	return nil
}

// validateNotBefore checks the nbf claim
func (t *IdentityToken) validateNotBefore(ctx context.Context, now time.Time) apperrors.Error {
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
	return nil
}

// validateIssuedAt checks the iat claim
func (t *IdentityToken) validateIssuedAt(ctx context.Context) apperrors.Error {
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
	return nil
}

// validateIssuer checks the iss claim
func (t *IdentityToken) validateIssuer(ctx context.Context) apperrors.Error {
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
	return nil
}

// validateAudience checks the aud claim
func (t *IdentityToken) validateAudience(ctx context.Context) apperrors.Error {
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
		if !t.containsExpectedAudience(v, expectedAudience) {
			log.Ctx(ctx).Debug().Interface("got", v).Str("expected", expectedAudience).Msg("invalid audience")
			return ErrInvalidToken.Msg(fmt.Sprintf("invalid audience: got %v, expected %s", v, expectedAudience))
		}
	default:
		log.Ctx(ctx).Debug().Type("type", v).Msg("invalid audience type")
		return ErrInvalidToken.Msg("invalid audience type")
	}
	return nil
}

// containsExpectedAudience checks if the expected audience is in the audience array
func (t *IdentityToken) containsExpectedAudience(audiences []any, expected string) bool {
	for _, a := range audiences {
		if s, ok := a.(string); ok && s == expected {
			return true
		}
	}
	return false
}

// validateJWTID checks the jti claim
func (t *IdentityToken) validateJWTID(ctx context.Context) apperrors.Error {
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
func (t *IdentityToken) Get(key string) (any, bool) {
	if t.claims == nil {
		return nil, false
	}
	val, ok := t.claims[key]
	return val, ok
}

// GetString retrieves a string claim value from the token
func (t *IdentityToken) GetString(key string) (string, bool) {
	val, ok := t.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetTokenUse returns the token use type
func (t *IdentityToken) GetTokenUse() catcommon.TokenType {
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

// GetSubject returns the subject claim
func (t *IdentityToken) GetSubject() string {
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

// GetTenantID returns the tenant ID from the token
func (t *IdentityToken) GetTenantID() string {
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

// GetUUID retrieves a UUID claim value from the token
func (t *IdentityToken) GetUUID(key string) (uuid.UUID, bool) {
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

// GetExpiry returns the token's expiration time
func (t *IdentityToken) GetExpiry() time.Time {
	exp, ok := t.claims["exp"].(float64)
	if !ok {
		return time.Time{}
	}
	return time.Unix(int64(exp), 0)
}

// GetRawToken returns the raw token string
func (t *IdentityToken) GetRawToken() string {
	if t.token == nil {
		return ""
	}
	return t.token.Raw
}
