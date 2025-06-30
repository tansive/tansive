package catalogmanager

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

// Base catalog error
var (
	ErrCatalogError apperrors.Error = apperrors.New("catalog processing failed").SetStatusCode(http.StatusInternalServerError)
)

// Not found errors
var (
	ErrCatalogNotFound   apperrors.Error = ErrCatalogError.New("catalog not found").SetExpandError(true).SetStatusCode(http.StatusNotFound)
	ErrObjectNotFound    apperrors.Error = ErrCatalogError.New("object not found").SetStatusCode(http.StatusNotFound)
	ErrVariantNotFound   apperrors.Error = ErrCatalogError.New("variant not found").SetStatusCode(http.StatusNotFound)
	ErrNamespaceNotFound apperrors.Error = ErrCatalogError.New("namespace not found").SetStatusCode(http.StatusNotFound)
	ErrViewNotFound      apperrors.Error = ErrCatalogError.New("view not found").SetStatusCode(http.StatusNotFound)
	ErrResourceNotFound  apperrors.Error = ErrCatalogError.New("resource not found").SetStatusCode(http.StatusNotFound)
)

// Ops errors
var (
	ErrUnableToLoadObject   apperrors.Error = ErrCatalogError.New("failed to load object").SetStatusCode(http.StatusInternalServerError)
	ErrUnableToUpdateObject apperrors.Error = ErrCatalogError.New("failed to update object").SetExpandError(true).SetStatusCode(http.StatusInternalServerError)
	ErrUnableToDeleteObject apperrors.Error = ErrCatalogError.New("failed to delete object").SetStatusCode(http.StatusInternalServerError)
)

// Conflict errors
var (
	ErrAlreadyExists         apperrors.Error = ErrCatalogError.New("object already exists").SetStatusCode(http.StatusConflict)
	ErrEqualToExistingObject apperrors.Error = ErrCatalogError.New("object is identical to existing object").SetStatusCode(http.StatusConflict)
)

// Validation errors
var (
	ErrEmptyMetadata             apperrors.Error = ErrCatalogError.New("metadata cannot be empty").SetStatusCode(http.StatusBadRequest)
	ErrInvalidProject            apperrors.Error = ErrCatalogError.New("invalid project").SetStatusCode(http.StatusBadRequest)
	ErrInvalidCatalog            apperrors.Error = ErrCatalogError.New("invalid catalog").SetStatusCode(http.StatusBadRequest)
	ErrInvalidVariant            apperrors.Error = ErrCatalogError.New("invalid variant").SetStatusCode(http.StatusBadRequest)
	ErrInvalidNamespace          apperrors.Error = ErrCatalogError.New("invalid namespace").SetStatusCode(http.StatusBadRequest)
	ErrInvalidObject             apperrors.Error = ErrCatalogError.New("invalid object").SetStatusCode(http.StatusBadRequest)
	ErrInvalidVersion            apperrors.Error = ErrCatalogError.New("invalid version").SetStatusCode(http.StatusBadRequest)
	ErrInvalidSession            apperrors.Error = ErrCatalogError.New("invalid session").SetStatusCode(http.StatusBadRequest)
	ErrInvalidUUID               apperrors.Error = ErrCatalogError.New("invalid UUID").SetStatusCode(http.StatusBadRequest)
	ErrInvalidRequest            apperrors.Error = ErrCatalogError.New("invalid request").SetStatusCode(http.StatusBadRequest)
	ErrInvalidView               apperrors.Error = ErrCatalogError.New("invalid view").SetStatusCode(http.StatusBadRequest)
	ErrInvalidSkillSetDefinition apperrors.Error = ErrCatalogError.New("invalid skillset definition").SetStatusCode(http.StatusBadRequest)
	ErrInvalidResourceValue      apperrors.Error = ErrCatalogError.New("invalid resource value").SetStatusCode(http.StatusBadRequest)
	ErrInvalidResourceDefinition apperrors.Error = ErrCatalogError.New("invalid resource definition").SetStatusCode(http.StatusBadRequest)
	ErrAmbiguousMatch            apperrors.Error = ErrCatalogError.New("ambiguous resource match").SetStatusCode(http.StatusBadRequest)
	ErrInvalidInput              apperrors.Error = ErrCatalogError.New("invalid input").SetStatusCode(http.StatusBadRequest)
)

// Schema validation errors
var (
	ErrSchemaValidation    apperrors.Error = apperrors.New("error validating schema").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrEmptySchema         apperrors.Error = ErrSchemaValidation.New("empty schema").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrSchemaSerialization apperrors.Error = ErrSchemaValidation.New("error serializing schema").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrInvalidSchema       apperrors.Error = ErrSchemaValidation.New("invalid schema").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrInvalidNameFormat   apperrors.Error = ErrSchemaValidation.New("invalid name format").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrValueValidation     apperrors.Error = ErrSchemaValidation.New("error validating value").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrInvalidType         apperrors.Error = ErrSchemaValidation.New("invalid type").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrInvalidKind         apperrors.Error = ErrSchemaValidation.New("unsupported kind").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
	ErrValueInvalid        apperrors.Error = ErrSchemaValidation.New("value failed validation").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
)

// Authorization errors
var (
	ErrAuthError                apperrors.Error = ErrCatalogError.New("authorization error").SetStatusCode(http.StatusForbidden)
	ErrUnauthorizedToCreateView apperrors.Error = ErrAuthError.New("unauthorized to create view").SetStatusCode(http.StatusForbidden)
	ErrDisallowedByPolicy       apperrors.Error = ErrAuthError.New("not allowed by policy").SetStatusCode(http.StatusForbidden)
)
