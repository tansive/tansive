package session

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

var (
	// ErrSessionError is the base error for all session-related errors.
	// Occurs when there is a general error in session processing.
	ErrSessionError apperrors.Error = apperrors.New("error in processing session").SetStatusCode(http.StatusInternalServerError)

	// ErrInvalidSession is returned when a session is invalid or not found.
	// Occurs when session ID is invalid or session does not exist.
	ErrInvalidSession apperrors.Error = ErrSessionError.New("invalid session").SetStatusCode(http.StatusBadRequest)

	// ErrAlreadyExists is returned when attempting to create a session that already exists.
	// Occurs when a session with the same ID already exists in the session manager.
	ErrAlreadyExists apperrors.Error = ErrSessionError.New("session already exists").SetStatusCode(http.StatusConflict)

	// ErrBadRequest is returned for malformed or invalid requests.
	// Occurs when request parameters are invalid or missing required fields.
	ErrBadRequest apperrors.Error = ErrSessionError.New("bad request").SetStatusCode(http.StatusBadRequest)

	// ErrChannelFailed is returned when communication channels fail.
	// Occurs when there are issues with event bus or communication channels.
	ErrChannelFailed apperrors.Error = ErrSessionError.New("channel failed").SetStatusCode(http.StatusInternalServerError)

	// ErrUnknownMethod is returned for unsupported HTTP methods.
	// Occurs when the requested HTTP method is not supported by the endpoint.
	ErrUnknownMethod apperrors.Error = ErrSessionError.New("unknown method").SetStatusCode(http.StatusMethodNotAllowed)

	// ErrInvalidParams is returned when function parameters are invalid.
	// Occurs when skill parameters or function arguments are malformed.
	ErrInvalidParams apperrors.Error = ErrSessionError.New("invalid parameters").SetStatusCode(http.StatusBadRequest)

	// ErrExecutionFailed is returned when skill execution fails.
	// Occurs when there is an error during skill runtime execution.
	ErrExecutionFailed apperrors.Error = ErrSessionError.New("execution failed").SetStatusCode(http.StatusInternalServerError)

	// ErrUnableToGetSkillset is returned when skillset retrieval fails.
	// Occurs when the catalog server is unavailable or skillset is not found.
	ErrUnableToGetSkillset apperrors.Error = ErrSessionError.New("unable to get skillset").SetStatusCode(http.StatusInternalServerError)

	// ErrUnableToGetViewDefinition is returned when view definition retrieval fails.
	// Occurs when the catalog server is unavailable or view definition is not found.
	ErrUnableToGetViewDefinition apperrors.Error = ErrSessionError.New("unable to get view definition").SetStatusCode(http.StatusInternalServerError)

	// ErrInvalidObject is returned when an object is invalid or malformed.
	// Occurs when JSON objects or data structures are invalid.
	ErrInvalidObject apperrors.Error = ErrSessionError.New("invalid object").SetStatusCode(http.StatusBadRequest)

	// ErrToolGraphError is returned when tool graph operations fail.
	// Occurs when there are issues with call graph management or tool invocation tracking.
	ErrToolGraphError apperrors.Error = ErrSessionError.New("tool graph error").SetStatusCode(http.StatusBadRequest)

	// ErrInvalidInvocationID is returned when an invocation ID is invalid or not found.
	// Occurs when the provided invocation ID does not exist in the current session.
	ErrInvalidInvocationID apperrors.Error = ErrSessionError.New("invalid invocation ID").SetStatusCode(http.StatusBadRequest)

	// ErrBlockedByPolicy is returned when policy enforcement blocks an action.
	// Occurs when the current view does not authorize the required actions for the skill.
	ErrBlockedByPolicy apperrors.Error = ErrSessionError.New("blocked by policy").SetStatusCode(http.StatusForbidden)

	// ErrTokenRequired is returned when authentication token is missing.
	// Occurs when a valid authentication token is required but not provided.
	ErrTokenRequired apperrors.Error = ErrSessionError.New("token is required").SetStatusCode(http.StatusBadRequest)

	// ErrTokenExpired is returned when authentication token has expired.
	// Occurs when the provided token has passed its expiration time.
	ErrTokenExpired apperrors.Error = ErrSessionError.New("token has expired").SetStatusCode(http.StatusBadRequest)

	// ErrFailedRequestToTansiveServer is returned when communication with Tansive server fails.
	// Occurs when HTTP requests to the catalog server fail or return errors.
	ErrFailedRequestToTansiveServer apperrors.Error = ErrSessionError.New("failed to make request to Tansive server").SetStatusCode(http.StatusInternalServerError)

	// ErrTransformUndefined is returned when a transform is referenced but not defined.
	// Occurs when a skill references a transform that is not available or properly configured.
	ErrTransformUndefined apperrors.Error = ErrSessionError.New("transform is undefined").SetStatusCode(http.StatusBadRequest)

	// ErrSkillNotMCP is returned when a skill is not an MCP server.
	// Occurs when a skill is not annotated with the MCP type.
	ErrSkillNotMCP apperrors.Error = ErrSessionError.New("skill is not an MCP server").SetStatusCode(http.StatusBadRequest)

	// ErrInvalidInput is returned when the input arguments are invalid.
	// Occurs when the input arguments are not a map[string]any.
	ErrInvalidInput apperrors.Error = ErrSessionError.New("invalid input arguments").SetStatusCode(http.StatusBadRequest)
)
