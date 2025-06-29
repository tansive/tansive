// Package api provides data structures for session API operations.
// Defines request and response types for session creation, retrieval, and listing operations.
// All types support JSON serialization for HTTP API communication.
package api

import (
	"github.com/tansive/tansive/pkg/types"
)

// Session represents a session with its identifier and context.
// Contains the session ID and optional context data for session operations.
type Session struct {
	ID      string            `json:"id"`      // unique session identifier
	Context types.NullableAny `json:"context"` // optional session context data
}

// CreateSessionRequest represents a request to create a new session.
// Contains session information required for session creation.
type CreateSessionRequest struct {
	Session
}

// CreateSessionResponse represents the response from session creation.
// Contains the ID of the newly created session.
type CreateSessionResponse struct {
	ID string `json:"id"` // unique identifier of the created session
}

// GetSessionResponse represents the response from session retrieval.
// Contains the complete session information including ID and context.
type GetSessionResponse struct {
	Session
}

// ListSessionsResponse represents the response from session listing.
// Contains an array of all available sessions.
type ListSessionsResponse struct {
	Sessions []Session `json:"sessions"` // array of session objects
}
