package auth

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

// Base auth error
var (
	ErrAuth apperrors.Error = apperrors.New("auth error").SetStatusCode(http.StatusInternalServerError)
)

// Not found errors
var (
	ErrCatalogNotFound apperrors.Error = ErrAuth.New("catalog not found").SetStatusCode(http.StatusNotFound)
	ErrViewNotFound    apperrors.Error = ErrAuth.New("view not found").SetStatusCode(http.StatusNotFound)
)

// Validation errors
var (
	ErrInvalidView      apperrors.Error = ErrAuth.New("invalid view").SetStatusCode(http.StatusBadRequest)
	ErrInvalidCatalog   apperrors.Error = ErrAuth.New("invalid catalog").SetStatusCode(http.StatusBadRequest)
	ErrInvalidRequest   apperrors.Error = ErrAuth.New("invalid request").SetStatusCode(http.StatusBadRequest)
	ErrInvalidViewRules apperrors.Error = ErrAuth.New("invalid view rules").SetStatusCode(http.StatusBadRequest)
	ErrMissingTenantID  apperrors.Error = ErrAuth.New("missing tenant ID").SetStatusCode(http.StatusBadRequest)
)

// Authorization errors
var (
	ErrUnauthorized       apperrors.Error = ErrAuth.New("unauthorized access").SetStatusCode(http.StatusUnauthorized)
	ErrInvalidToken       apperrors.Error = ErrAuth.New("invalid token").SetStatusCode(http.StatusUnauthorized)
	ErrUnableToParseToken apperrors.Error = ErrAuth.New("unable to parse token").SetStatusCode(http.StatusForbidden)
	ErrDisallowedByPolicy apperrors.Error = ErrAuth.New("disallowed by policy").SetStatusCode(http.StatusForbidden)
)

// Token errors
var (
	ErrTokenGeneration            apperrors.Error = ErrAuth.New("failed to generate token").SetStatusCode(http.StatusInternalServerError)
	ErrUnableToGenerateSigningKey apperrors.Error = ErrAuth.New("unable to generate signing key").SetStatusCode(http.StatusInternalServerError)
	ErrUnableToParseTokenDuration apperrors.Error = ErrAuth.New("unable to parse token duration").SetStatusCode(http.StatusInternalServerError)
)

// Ops errors
var (
	ErrUnableToCreateView apperrors.Error = ErrAuth.New("unable to create view").SetStatusCode(http.StatusInternalServerError)
	ErrBadRequest         apperrors.Error = ErrAuth.New("bad request").SetStatusCode(http.StatusBadRequest)
)
