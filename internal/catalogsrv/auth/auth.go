package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/httpx"
	"github.com/tansive/tansive/internal/common/uuid"
)

// adoptViewRsp represents the response structure for view adoption operations
type adoptViewRsp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// getCatalogByRef retrieves a catalog by either its ID or name
func getCatalogByRef(ctx context.Context, catalogRef string) (*models.Catalog, apperrors.Error) {
	catalogID, err := uuid.Parse(catalogRef)
	if err != nil {
		return db.DB(ctx).GetCatalogByName(ctx, catalogRef)
	}
	return db.DB(ctx).GetCatalogByID(ctx, catalogID)
}

// adoptView adopts a view from a catalog. The parent view must be scoped to the catalog and
// the derived view must have a policy subset of the parent view.
func adoptView(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	catalogRef := chi.URLParam(r, "catalogRef")
	viewLabel := chi.URLParam(r, "viewLabel")

	catalog, err := getCatalogByRef(ctx, catalogRef)
	if err != nil {
		return nil, ErrCatalogNotFound.Err(err)
	}

	// Validate current context
	ourViewDef := policy.GetViewDefinition(ctx)
	if ourViewDef == nil {
		return nil, ErrInvalidView.Msg("no current view definition found")
	}
	if ourViewDef.Scope.Catalog != catalog.Name {
		return nil, ErrInvalidView.Msg("current view not in catalog: " + catalog.Name)
	}

	// Check if our current view has permission to adopt the view
	allowed, err := policy.CanAdoptView(ctx, viewLabel)
	if err != nil {
		return nil, err
	}

	// If this is a user, check if they have permission to adopt the view
	if !allowed {
		allowed = policy.CanAdoptViewAsUser(ctx, viewLabel)
	}

	if !allowed {
		return nil, ErrDisallowedByPolicy.Msg("view is not allowed to be adopted")
	}

	wantView, err := db.DB(ctx).GetViewByLabel(ctx, viewLabel, catalog.CatalogID)
	if err != nil {
		return nil, ErrViewNotFound.Err(err)
	}

	token, tokenExpiry, err := CreateAccessToken(ctx,
		wantView,
		WithAdditionalClaims(getAccessTokenClaims(ctx)),
	)
	if err != nil {
		return nil, ErrTokenGeneration.Msg(err.Error())
	}

	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response: &adoptViewRsp{
			Token:     token,
			ExpiresAt: tokenExpiry,
		},
	}, nil
}

// adoptDefaultCatalogView adopts the default view for a catalog.
func adoptDefaultCatalogView(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	catalogRef := chi.URLParam(r, "catalogRef")

	catalog, err := getCatalogByRef(ctx, catalogRef)
	if err != nil {
		return nil, ErrCatalogNotFound.Err(err)
	}

	wantView, err := getDefaultUserViewDefInCatalog(ctx, catalog.CatalogID)
	if err != nil {
		return nil, err
	}

	userContext := catcommon.GetUserContext(ctx)
	if userContext == nil || userContext.UserID == "" {
		return nil, ErrUnauthorized
	}

	token, tokenExpiry, err := CreateAccessToken(ctx,
		wantView,
		WithAdditionalClaims(getAccessTokenClaims(ctx)),
	)
	if err != nil {
		return nil, ErrTokenGeneration.Err(err)
	}

	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response: &adoptViewRsp{
			Token:     token,
			ExpiresAt: tokenExpiry,
		},
	}, nil
}

// getDefaultUserViewDefInCatalog retrieves the default view definition for a user in a catalog.
func getDefaultUserViewDefInCatalog(ctx context.Context, catalogID uuid.UUID) (*models.View, apperrors.Error) {
	userContext := catcommon.GetUserContext(ctx)
	if userContext == nil || userContext.UserID == "" {
		return nil, ErrUnauthorized
	}

	// Currently in single user mode, return admin view
	v, err := db.DB(ctx).GetViewByLabel(ctx, catcommon.DefaultAdminViewLabel, catalogID)
	if err != nil {
		return nil, ErrViewNotFound.Err(err)
	}
	return v, nil
}

func getAccessTokenClaims(ctx context.Context) map[string]any {
	var subject string
	userContext := catcommon.GetUserContext(ctx)
	if userContext == nil || userContext.UserID == "" {
		return nil
	}
	subject = "user/" + userContext.UserID

	return map[string]any{
		"token_use": catcommon.AccessTokenType,
		"sub":       subject,
	}
}
