package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"

	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/uuid"
)

func setupTest(t *testing.T) (context.Context, catcommon.TenantId, catcommon.ProjectId, uuid.UUID, uuid.UUID, *config.ConfigParam) {
	// Initialize context with logger and database connection
	ctx := newDb()

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant and project for testing
	err := db.DB(ctx).CreateTenant(ctx, tenantID)
	require.NoError(t, err)

	err = db.DB(ctx).CreateProject(ctx, projectID)
	require.NoError(t, err)

	// Register cleanup function that will run even if test panics
	t.Cleanup(func() {
		db.DB(ctx).DeleteProject(ctx, projectID)
		db.DB(ctx).DeleteTenant(ctx, tenantID)
		db.DB(ctx).Close(ctx)
	})

	// Set up test configuration
	cfg := config.Config()
	cfg.Auth.DefaultTokenValidity = "1h"
	cfg.ServerHostName = "local.tansive.dev"
	cfg.ServerPort = "8678"
	cfg.Auth.KeyEncryptionPasswd = "test-password"

	// Create a catalog for testing
	var info pgtype.JSONB
	err = info.Set(map[string]interface{}{"meta": "test"})
	require.NoError(t, err)

	catalogID := uuid.New()
	catalog := &models.Catalog{
		CatalogID:   catalogID,
		Name:        "test-catalog",
		Description: "Test catalog",
		ProjectID:   projectID,
		Info:        info,
	}
	err = db.DB(ctx).CreateCatalog(ctx, catalog)
	require.NoError(t, err)

	// Create parent view
	parentView := &policy.ViewDefinition{
		Scope: policy.Scope{
			Catalog: "test-catalog",
		},
		Rules: policy.Rules{
			{
				Intent:  policy.IntentAllow,
				Actions: []policy.Action{policy.ActionCatalogList, policy.ActionVariantList},
				Targets: []policy.TargetResource{"res://catalogs/test-catalog"},
			},
		},
	}

	// Convert parent view to JSON
	parentViewJSON, err := json.Marshal(parentView)
	require.NoError(t, err)

	// Create the view model
	view := &models.View{
		Label:       "parent-view",
		Description: "Parent view for testing",
		Rules:       parentViewJSON,
		CatalogID:   catalogID,
		TenantID:    tenantID,
		CreatedBy:   "user/test_user",
		UpdatedBy:   "user/test_user",
	}

	// Store the parent view in the database
	err = db.DB(ctx).CreateView(ctx, view)
	require.NoError(t, err)

	log.Ctx(ctx).Info().
		Str("view_id", view.ViewID.String()).
		Str("catalog_id", catalogID.String()).
		Msg("Created view successfully")

	// Create an active signing key
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	encKey, err := catcommon.Encrypt(priv, cfg.Auth.KeyEncryptionPasswd)
	require.NoError(t, err)

	signingKey := &models.SigningKey{
		PublicKey:  pub,
		PrivateKey: encKey,
		IsActive:   true,
	}
	err = db.DB(ctx).CreateSigningKey(ctx, signingKey)
	require.NoError(t, err)

	return ctx, tenantID, projectID, view.ViewID, catalogID, cfg
}

func TestCreateToken(t *testing.T) {
	ctx, _, _, testViewID, catalogID, _ := setupTest(t)

	t.Run("successful token creation", func(t *testing.T) {
		// Create and save the derived view first
		derivedViewDef := &policy.ViewDefinition{
			Scope: policy.Scope{
				Catalog: "test-catalog",
			},
			Rules: policy.Rules{
				{
					Intent:  policy.IntentAllow,
					Actions: []policy.Action{policy.ActionCatalogList},
					Targets: []policy.TargetResource{"res://catalogs/test-catalog"},
				},
			},
		}

		// Convert derived view to JSON
		derivedViewJSON, err := json.Marshal(derivedViewDef)
		require.NoError(t, err)

		// Create the derived view model
		derivedView := &models.View{
			Label:       "derived-view",
			Description: "Derived view for testing",
			Rules:       derivedViewJSON,
			Info:        nil,
			CatalogID:   catalogID,
			TenantID:    catcommon.TenantId("TABCDE"),
			CreatedBy:   "user/test_user",
			UpdatedBy:   "user/test_user",
		}

		// Store the derived view in the database
		err = db.DB(ctx).CreateView(ctx, derivedView)
		require.NoError(t, err)

		// Get the parent view definition
		parentView, err := db.DB(ctx).GetView(ctx, testViewID)
		require.NoError(t, err)
		parentViewDef := &policy.ViewDefinition{}
		err = json.Unmarshal(parentView.Rules, &parentViewDef)
		require.NoError(t, err)

		// Create token with parent view definition option
		token, expiry, appErr := CreateAccessToken(ctx, derivedView, WithParentViewDefinition(parentViewDef))
		require.NoError(t, appErr)
		require.NotEmpty(t, token)
		require.False(t, expiry.IsZero())

		// Get the signing key from database
		dbKey, dbErr := db.DB(ctx).GetActiveSigningKey(ctx)
		require.NoError(t, dbErr)
		require.NotNil(t, dbKey)

		// Parse and verify the token
		parsedToken, parseErr := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return ed25519.PublicKey(dbKey.PublicKey), nil
		})
		require.NoError(t, parseErr)
		require.True(t, parsedToken.Valid)

		// Extract claims
		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		require.True(t, ok)

		// Verify token ID and view ID
		jti, ok := claims["jti"].(string)
		require.True(t, ok)
		require.NotEmpty(t, jti)

		viewID, ok := claims["view_id"].(string)
		require.True(t, ok)
		require.Equal(t, derivedView.ViewID.String(), viewID)

		// Verify token is stored in database
		storedToken, dbErr := db.DB(ctx).GetViewToken(ctx, uuid.MustParse(jti))
		require.NoError(t, dbErr)
		require.Equal(t, derivedView.ViewID, storedToken.ViewID)
	})

	t.Run("invalid view", func(t *testing.T) {
		token, expiry, err := CreateAccessToken(ctx, nil)
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.True(t, expiry.IsZero())
	})

	// t.Run("missing parent view", func(t *testing.T) {
	// 	derivedView := &models.View{
	// 		Label:       "derived-view",
	// 		Description: "Derived view for testing",
	// 		Rules:       []byte(`{"scope":{"catalog":"test-catalog"},"rules":[{"intent":"allow","actions":["catalog:list"],"targets":["res://catalogs/test-catalog"]}]}`),
	// 		CatalogID:   catalogID,
	// 		TenantID:    catcommon.TenantId("TABCDE"),
	// 	}

	// 	token, expiry, err := CreateToken(ctx, derivedView)
	// 	assert.Error(t, err)
	// 	assert.Empty(t, token)
	// 	assert.True(t, expiry.IsZero())
	// })
}
