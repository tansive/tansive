package session

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

var (
	ErrSessionError       apperrors.Error = apperrors.New("session error")
	ErrInvalidSession     apperrors.Error = ErrSessionError.New("invalid session").SetStatusCode(http.StatusBadRequest)
	ErrInvalidObject      apperrors.Error = ErrSessionError.New("invalid object").SetStatusCode(http.StatusBadRequest)
	ErrInvalidView        apperrors.Error = ErrSessionError.New("invalid view").SetStatusCode(http.StatusBadRequest)
	ErrInvalidViewDef     apperrors.Error = ErrSessionError.New("invalid view definition").SetStatusCode(http.StatusBadRequest)
	ErrDisallowedByPolicy apperrors.Error = ErrSessionError.New("disallowed by policy").SetStatusCode(http.StatusForbidden)
	ErrNotAuthorized      apperrors.Error = ErrSessionError.New("not authorized").SetStatusCode(http.StatusForbidden)
	ErrInvalidRequest     apperrors.Error = ErrSessionError.New("invalid request").SetStatusCode(http.StatusBadRequest)
	ErrUnableToGetSession apperrors.Error = ErrSessionError.New("unable to get session").SetStatusCode(http.StatusBadRequest)
)
