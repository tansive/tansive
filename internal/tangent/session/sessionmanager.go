package session

import (
	"context"
	"time"

	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
)

// SessionManager defines the interface for session lifecycle management.
// Provides methods for creating, retrieving, listing, and deleting sessions.
// All methods require valid context and return application errors on failure.
type SessionManager interface {
	// CreateSession creates a new session with the given context and authentication token.
	// Returns the created session and any error encountered during creation.
	CreateSession(context.Context, *ServerContext, string, time.Time, tangentcommon.SessionType) (*session, apperrors.Error)

	// GetSession retrieves a session by its unique identifier.
	// Returns the session and any error encountered during retrieval.
	GetSession(uuid.UUID) (*session, apperrors.Error)

	// ListSessions returns all active sessions in the session manager.
	// Returns the session list and any error encountered during listing.
	ListSessions() ([]*session, apperrors.Error)

	// DeleteSession removes a session from the session manager.
	// Cleans up associated resources and event bus subscriptions.
	DeleteSession(uuid.UUID) apperrors.Error
}
