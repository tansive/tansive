package apis

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/httpx"
)

// ToHTTPXError converts an application error to an HTTP error with appropriate status code and description.
// If the error is not an application error, it returns the original error unchanged.
func ToHTTPXError(err error) error {
	if appErr, ok := err.(apperrors.Error); ok {
		statusCode := appErr.StatusCode()
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		return &httpx.Error{
			StatusCode:  statusCode,
			Description: appErr.ErrorAll(),
		}
	}
	return err
}

var (
	ErrBadRequest      apperrors.Error = apperrors.New("Bad Request").SetStatusCode(http.StatusBadRequest)
	ErrBlockedByPolicy apperrors.Error = ErrBadRequest.New("blocked by policy").SetStatusCode(http.StatusForbidden)
	ErrEmptyCatalog    apperrors.Error = ErrBadRequest.New("no catalog provided").SetStatusCode(http.StatusBadRequest)
)
