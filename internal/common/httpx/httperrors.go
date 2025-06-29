package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

// Error represents an HTTP error response with status code and description.
type Error struct {
	Description string `json:"description"`
	StatusCode  int    `json:"http_status_code"`
}

type errorRsp struct {
	Result int    `json:"result"`
	Error  string `json:"error"`
}

// Failure represents the error result code in error responses.
const Failure int = 0

// Send writes the error response to the provided ResponseWriter.
// If the writer is nil, no action is taken.
func (e *Error) Send(w http.ResponseWriter) {
	if w != nil {
		rsp := &errorRsp{
			Result: Failure,
			Error:  e.Description,
		}
		// Encode the response struct as JSON and send it
		rspJson, err := json.Marshal(rsp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unable to parse error"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(e.StatusCode)
		w.Write(rspJson)
	}
}

// Error returns the error description.
func (e *Error) Error() string {
	return e.Description
}

// Is reports whether the error matches the target error.
func (current Error) Is(other error) bool {
	return current.Error() == other.Error()
}

// SendError sends an application error as an HTTP error response.
// If the error is nil, no action is taken.
func SendError(w http.ResponseWriter, err apperrors.Error) {
	if err == nil {
		return
	}
	statusCode := err.StatusCode()
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	httperror := &Error{
		StatusCode:  statusCode,
		Description: err.ErrorAll(),
	}
	httperror.Send(w)
}

// Common Errors

// ErrPostReqNotSupported returns an error for unsupported POST requests.
func ErrPostReqNotSupported() *Error {
	return &Error{
		Description: "post request not supported",
		StatusCode:  http.StatusMethodNotAllowed,
	}
}

// ErrGetReqNotSupported returns an error for unsupported GET requests.
func ErrGetReqNotSupported() *Error {
	return &Error{
		Description: "get request not supported",
		StatusCode:  http.StatusMethodNotAllowed,
	}
}

// ErrReqMethodNotSupported returns an error for unsupported HTTP methods.
func ErrReqMethodNotSupported() *Error {
	return &Error{
		Description: "request method not supported",
		StatusCode:  http.StatusMethodNotAllowed,
	}
}

// ErrUnableToParseReqData returns an error when request data cannot be parsed.
func ErrUnableToParseReqData() *Error {
	return &Error{
		Description: "unable to parse request data",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrUnableToReadRequest returns an error when request data cannot be read.
func ErrUnableToReadRequest() *Error {
	return &Error{
		Description: "unable to read request data",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrApplicationError returns an error for application-level failures.
// If no message is provided, a default message is used.
func ErrApplicationError(err ...string) *Error {
	var s string
	if len(err) > 0 {
		s = err[0]
	} else {
		s = "unable to process request"
	}
	return &Error{
		Description: s,
		StatusCode:  http.StatusInternalServerError,
	}
}

// ErrUnAuthorized returns an error for unauthorized requests.
// If no message is provided, a default message is used.
func ErrUnAuthorized(str ...string) *Error {
	var s string
	if len(str) > 0 {
		s = str[0]
	} else {
		s = "unable to authenticate request"
	}
	return &Error{
		Description: s,
		StatusCode:  http.StatusUnauthorized,
	}
}

// ErrMissingKeyInRequest returns an error when authentication key is missing.
func ErrMissingKeyInRequest() *Error {
	return &Error{
		Description: "missing authentication key in request",
		StatusCode:  http.StatusUnauthorized,
	}
}

// ErrInvalidRequest returns an error for invalid request data.
// If no message is provided, a default message is used.
func ErrInvalidRequest(str ...string) *Error {
	var s string
	if len(str) > 0 {
		s = str[0]
	} else {
		s = "invalid request data or empty request values"
	}
	return &Error{
		Description: s,
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidTenantId returns an error for invalid tenant ID.
func ErrInvalidTenantId() *Error {
	return &Error{
		Description: "invalid tenant id",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidProjectId returns an error for invalid project ID.
func ErrInvalidProjectId() *Error {
	return &Error{
		Description: "invalid project id",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidNodeId returns an error for invalid node ID.
func ErrInvalidNodeId() *Error {
	return &Error{
		Description: "invalid node id",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidCatalog returns an error for invalid catalog.
func ErrInvalidCatalog() *Error {
	return &Error{
		Description: "invalid catalog",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidView returns an error for invalid view.
// If no message is provided, a default message is used.
func ErrInvalidView(err ...string) *Error {
	var s string
	if len(err) > 0 {
		s = err[0]
	} else {
		s = "invalid view"
	}
	return &Error{
		Description: s,
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidVariant returns an error for invalid variant.
func ErrInvalidVariant() *Error {
	return &Error{
		Description: "invalid variant",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidNamespace returns an error for invalid namespace.
func ErrInvalidNamespace() *Error {
	return &Error{
		Description: "invalid namespace",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidWorkspace returns an error for invalid workspace.
func ErrInvalidWorkspace() *Error {
	return &Error{
		Description: "invalid workspace",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidOnboardingKey returns an error for invalid onboarding key.
func ErrInvalidOnboardingKey() *Error {
	return &Error{
		Description: "invalid onboarding key",
		StatusCode:  http.StatusBadRequest,
	}
}

// ErrInvalidUser returns an error for invalid user.
func ErrInvalidUser() *Error {
	return &Error{
		Description: "invalid user",
		StatusCode:  http.StatusUnauthorized,
	}
}

// ErrUnableToServeRequest returns an error when request cannot be served.
func ErrUnableToServeRequest() *Error {
	return &Error{
		Description: "unable to serve request",
		StatusCode:  http.StatusInternalServerError,
	}
}

// ErrRequestTimeout returns an error for request timeout.
func ErrRequestTimeout() *Error {
	return &Error{
		Description: "request timed out",
		StatusCode:  http.StatusRequestTimeout,
	}
}

// ErrRequestTooLarge returns an error when request body exceeds size limit.
func ErrRequestTooLarge(limit int64) *Error {
	return &Error{
		Description: fmt.Sprintf("request body too large (limit: %d bytes)", limit),
		StatusCode:  http.StatusRequestEntityTooLarge,
	}
}
