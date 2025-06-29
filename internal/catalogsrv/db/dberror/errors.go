package dberror

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

var (
	ErrDatabase                  apperrors.Error = apperrors.New("db error").SetStatusCode(http.StatusInternalServerError)
	ErrAlreadyExists             apperrors.Error = ErrDatabase.New("already exists").SetStatusCode(http.StatusConflict)
	ErrNotFound                  apperrors.Error = ErrDatabase.New("not found").SetStatusCode(http.StatusNotFound)
	ErrInvalidInput              apperrors.Error = ErrDatabase.New("invalid input").SetStatusCode(http.StatusBadRequest)
	ErrInvalidCatalog            apperrors.Error = ErrDatabase.New("invalid catalog").SetStatusCode(http.StatusBadRequest)
	ErrInvalidVariant            apperrors.Error = ErrDatabase.New("invalid variant").SetStatusCode(http.StatusBadRequest)
	ErrMissingTenantID           apperrors.Error = ErrInvalidInput.New("missing tenant ID").SetStatusCode(http.StatusBadRequest)
	ErrMissingProjecID           apperrors.Error = ErrInvalidInput.New("missing project ID").SetStatusCode(http.StatusBadRequest)
	ErrMissingUserContext        apperrors.Error = ErrInvalidInput.New("missing user context").SetStatusCode(http.StatusBadRequest)
	ErrNoAncestorReferencesFound apperrors.Error = ErrDatabase.New("no ancestor references found").SetStatusCode(http.StatusBadRequest)
)
