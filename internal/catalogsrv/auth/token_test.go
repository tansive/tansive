package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/auth/keymanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestNewToken(t *testing.T) {
	ctx, _, _, viewID, _, _ := setupTest(t)
	serverAddr := config.Config().ServerHostName + ":" + config.Config().ServerPort

	claims := jwt.MapClaims{
		"view_id":   viewID.String(),
		"tenant_id": "TABCDE",
		"iss":       serverAddr,
		"aud":       []string{"tansivesrv"},
		"jti":       uuid.New().String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"nbf":       time.Now().Unix(),
		"sub":       "test-subject",
		"token_use": "access",
		"ver":       string(catcommon.TokenVersionV0_1),
	}

	signingKey, err := keymanager.GetKeyManager().GetActiveKey(ctx)
	require.NoError(t, err)
	require.NotNil(t, signingKey)

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	var tokenString string
	var goerr error
	tokenString, goerr = token.SignedString(signingKey.PrivateKey)
	if goerr != nil {
		err = ErrUnableToParseToken.MsgErr("unable to sign token", goerr)
	}
	require.NoError(t, err)

	tests := []struct {
		name        string
		tokenString string
		wantErr     bool
	}{
		{
			name:        "Valid token",
			tokenString: tokenString,
			wantErr:     false,
		},
		{
			name:        "Invalid token string",
			tokenString: "invalid.token.string",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenType, jwtToken, err := ParseAndValidateToken(ctx, tt.tokenString)
			if err != nil {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, catcommon.AccessTokenType, tokenType)
			assert.NotNil(t, jwtToken)
			token, err := ResolveAccessToken(ctx, jwtToken)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, token)
		})
	}
}

func TestTokenValidation(t *testing.T) {
	ctx, _, _, viewID, _, _ := setupTest(t)
	now := time.Now()
	serverAddr := config.Config().ServerHostName + ":" + config.Config().ServerPort

	tests := []struct {
		name    string
		claims  jwt.MapClaims
		isValid bool
	}{
		{
			name: "Valid token",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: true,
		},
		{
			name: "Missing version claim",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
			},
			isValid: false,
		},
		{
			name: "Invalid version claim",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{serverAddr},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       "0.2", // Wrong version
			},
			isValid: false,
		},
		{
			name: "Invalid version claim type",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       0.1, // Wrong type (float instead of string)
			},
			isValid: false,
		},
		{
			name: "Expired token",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(-time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token not yet valid (nbf)",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Add(time.Hour).Unix(), // Token not valid for another hour
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token issued too far in the past (iat)",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Add(-25 * time.Hour).Unix(), // Issued more than 24 hours ago
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with missing required claims",
			claims: jwt.MapClaims{
				"view_id": viewID.String(),
				"exp":     now.Add(time.Hour).Unix(),
			},
			isValid: false,
		},
		{
			name: "Token with invalid audience type",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       "not-an-array",
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with wrong audience",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"wrong-audience"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with wrong issuer",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       "wrong-issuer",
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with missing jti",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with multiple audiences including correct one",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"wrong-audience", "tansivesrv", "another-wrong"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: true,
		},
		{
			name: "Token just expired within skew window",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(-config.Config().Auth.GetClockSkewOrDefault() / 2).Unix(), // Expired but within skew
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: true,
		},
		{
			name: "Token just expired outside skew window",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(-config.Config().Auth.GetClockSkewOrDefault() * 2).Unix(), // Expired outside skew
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token not yet valid within skew window",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Add(config.Config().Auth.GetClockSkewOrDefault() / 2).Unix(), // Not yet valid but within skew
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: true,
		},
		{
			name: "Token not yet valid outside skew window",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Add(config.Config().Auth.GetClockSkewOrDefault() * 2).Unix(), // Not yet valid outside skew
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with string audience",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       "tansivesrv",
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: true,
		},
		{
			name: "Token with empty audience array",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with invalid exp type",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       "not-a-number",
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with invalid iat type",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       true,
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with invalid nbf type",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       uuid.New().String(),
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       []string{"not-a-number"},
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
		{
			name: "Token with invalid jti type",
			claims: jwt.MapClaims{
				"view_id":   viewID.String(),
				"tenant_id": "TABCDE",
				"iss":       serverAddr,
				"aud":       []string{"tansivesrv"},
				"jti":       123,
				"exp":       now.Add(time.Hour).Unix(),
				"iat":       now.Unix(),
				"nbf":       now.Unix(),
				"ver":       string(catcommon.TokenVersionV0_1),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signingKey, err := keymanager.GetKeyManager().GetActiveKey(ctx)
			require.NoError(t, err)
			require.NotNil(t, signingKey)

			token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, tt.claims)
			var tokenString string
			var goerr error
			tokenString, goerr = token.SignedString(signingKey.PrivateKey)
			if goerr != nil {
				err = ErrTokenGeneration.MsgErr("unable to sign token", goerr)
			}
			require.NoError(t, err)

			// For expired tokens, we need to disable the JWT library's built-in validation
			parsedToken, parseErr := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return signingKey.PublicKey, nil
			}, jwt.WithoutClaimsValidation())

			if !tt.isValid {
				if parseErr != nil {
					return // Expected error
				}
				// If no error, token should still be invalid according to our validation
				tokenObj := &Token{
					token:  parsedToken,
					claims: parsedToken.Claims.(jwt.MapClaims),
				}
				validateErr := tokenObj.Validate(ctx)
				assert.Error(t, validateErr)
				return
			}

			require.NoError(t, parseErr)
			tokenObj := &Token{
				token:  parsedToken,
				claims: parsedToken.Claims.(jwt.MapClaims),
			}
			validateErr := tokenObj.Validate(ctx)
			if tt.isValid {
				assert.NoError(t, validateErr)
			} else {
				assert.Error(t, validateErr)
			}
		})
	}
}

func TestTokenGetters(t *testing.T) {
	ctx, _, _, viewID, _, _ := setupTest(t)
	subject := "test-subject"
	tenantID := "TABCDE"
	tokenUse := "access"
	serverAddr := config.Config().ServerHostName + ":" + config.Config().ServerPort

	claims := jwt.MapClaims{
		"sub":       subject,
		"tenant_id": tenantID,
		"token_use": tokenUse,
		"view_id":   viewID.String(),
		"iss":       serverAddr,
		"aud":       []string{"tansivesrv"},
		"jti":       uuid.New().String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"nbf":       time.Now().Unix(),
		"ver":       string(catcommon.TokenVersionV0_1),
	}

	signingKey, err := keymanager.GetKeyManager().GetActiveKey(ctx)
	require.NoError(t, err)
	require.NotNil(t, signingKey)

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	var tokenString string
	var goerr error
	tokenString, goerr = token.SignedString(signingKey.PrivateKey)
	if goerr != nil {
		err = ErrTokenGeneration.MsgErr("unable to sign token", goerr)
	}
	require.NoError(t, err)

	tokenType, jwtToken, err := ParseAndValidateToken(ctx, tokenString)
	if err != nil {
		assert.Error(t, err)
		return
	}
	assert.Equal(t, catcommon.AccessTokenType, tokenType)
	assert.NotNil(t, jwtToken)
	parsedToken, err := ResolveAccessToken(ctx, jwtToken)
	require.NoError(t, err)

	t.Run("GetSubject", func(t *testing.T) {
		assert.Equal(t, subject, parsedToken.GetSubject())
	})

	t.Run("GetTenantID", func(t *testing.T) {
		assert.Equal(t, tenantID, parsedToken.GetTenantID())
	})

	t.Run("GetTokenUse", func(t *testing.T) {
		assert.Equal(t, catcommon.TokenType(tokenUse), parsedToken.GetTokenUse())
	})

	t.Run("GetViewID", func(t *testing.T) {
		assert.Equal(t, viewID, parsedToken.GetViewID())
	})

	t.Run("GetView", func(t *testing.T) {
		assert.Equal(t, viewID, parsedToken.GetView().ViewID)
	})

	t.Run("GetUUID", func(t *testing.T) {
		id, ok := parsedToken.GetUUID("view_id")
		assert.True(t, ok)
		assert.Equal(t, viewID, id)
	})
}

func TestParseAndValidateToken(t *testing.T) {
	ctx, tenantID, _, viewID, _, _ := setupTest(t)

	// Create a token
	signingKey, err := keymanager.GetKeyManager().GetActiveKey(ctx)
	require.NoError(t, err)
	require.NotNil(t, signingKey)

	serverAddr := config.Config().ServerHostName + ":" + config.Config().ServerPort
	claims := jwt.MapClaims{
		"view_id":   viewID.String(),
		"tenant_id": string(tenantID),
		"iss":       serverAddr,
		"aud":       []string{"tansivesrv"},
		"jti":       uuid.New().String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"nbf":       time.Now().Unix(),
		"ver":       string(catcommon.TokenVersionV0_1),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenString, goerr := token.SignedString(signingKey.PrivateKey)
	if goerr != nil {
		t.Fatalf("Failed to sign token: %v", goerr)
	}

	// Test parsing and validation
	tokenType, jwtToken, err := ParseAndValidateToken(ctx, tokenString)
	if err != nil {
		assert.Error(t, err)
		return
	}
	assert.Equal(t, catcommon.AccessTokenType, tokenType)
	assert.NotNil(t, jwtToken)
	parsedToken, err := ResolveAccessToken(ctx, jwtToken)
	require.NoError(t, err)
	assert.NotNil(t, parsedToken)
	assert.Equal(t, viewID, parsedToken.view.ViewID)
	assert.Equal(t, tenantID, parsedToken.view.TenantID)
}
