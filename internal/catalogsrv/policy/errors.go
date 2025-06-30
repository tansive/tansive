package policy

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

// Base catalog error
var (
	ErrViewError apperrors.Error = apperrors.New("view processing failed").SetStatusCode(http.StatusInternalServerError)
)

// Not found errors
var (
	ErrCatalogNotFound   apperrors.Error = ErrViewError.New("catalog not found").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrObjectNotFound    apperrors.Error = ErrViewError.New("object not found").SetStatusCode(http.StatusBadRequest)
	ErrVariantNotFound   apperrors.Error = ErrViewError.New("variant not found").SetStatusCode(http.StatusBadRequest)
	ErrNamespaceNotFound apperrors.Error = ErrViewError.New("namespace not found").SetStatusCode(http.StatusBadRequest)
	ErrViewNotFound      apperrors.Error = ErrViewError.New("view not found").SetStatusCode(http.StatusBadRequest)
)

// Operation errors
var (
	ErrUnableToLoadObject   apperrors.Error = ErrViewError.New("failed to load object").SetStatusCode(http.StatusInternalServerError)
	ErrUnableToDeleteObject apperrors.Error = ErrViewError.New("failed to delete object").SetStatusCode(http.StatusInternalServerError)
)

// Conflict errors
var (
	ErrAlreadyExists apperrors.Error = ErrViewError.New("object already exists").SetStatusCode(http.StatusConflict)
)

// Validation errors
var (
	ErrInvalidProject  apperrors.Error = ErrViewError.New("invalid project").SetStatusCode(http.StatusBadRequest)
	ErrInvalidCatalog  apperrors.Error = ErrViewError.New("invalid catalog").SetStatusCode(http.StatusBadRequest)
	ErrInvalidView     apperrors.Error = ErrViewError.New("invalid view").SetStatusCode(http.StatusBadRequest)
	ErrInvalidSkillSet apperrors.Error = ErrViewError.New("invalid skillset").SetStatusCode(http.StatusBadRequest)
)

// Schema validation errors
var (
	ErrSchemaValidation apperrors.Error = ErrViewError.New("error validating schema").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrInvalidSchema    apperrors.Error = ErrSchemaValidation.New("invalid schema").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
)

// Authorization errors
var (
	ErrAuthError                apperrors.Error = ErrViewError.New("authorization error").SetStatusCode(http.StatusForbidden)
	ErrUnauthorizedToCreateView apperrors.Error = ErrAuthError.New("unauthorized to create view").SetStatusCode(http.StatusForbidden)
	ErrDisallowedByPolicy       apperrors.Error = ErrAuthError.New("not allowed by policy").SetStatusCode(http.StatusForbidden)
)

var (
// ErrViewPolicyError            apperrors.Error = ErrViewError.New("authorization error").SetStatusCode(http.StatusInternalServerError)
// ErrUnableToGenerateSigningKey apperrors.Error = ErrViewPolicyError.New("unable to generate signing key").SetStatusCode(http.StatusInternalServerError)
// ErrUnableToParseTokenDuration apperrors.Error = ErrViewPolicyError.New("unable to parse token duration").SetStatusCode(http.StatusInternalServerError)
// ErrUnableToGenerateToken      apperrors.Error = ErrViewPolicyError.New("unable to generate token").SetStatusCode(http.StatusInternalServerError)
// ErrUnableToCreateView         apperrors.Error = ErrViewPolicyError.New("unable to create view").SetStatusCode(http.StatusInternalServerError)
)
