package session

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/session/toolgraph"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
)

// activeSessions manages the collection of active sessions.
// Provides thread-safe access to session storage and lifecycle management.
type activeSessions struct {
	sessions map[uuid.UUID]*session
}

// ServerContext defines the execution context for a session.
// Contains all necessary information for skill execution and policy evaluation.
type ServerContext struct {
	SessionID        uuid.UUID              `json:"session_id"`        // unique session identifier
	SkillSet         string                 `json:"skillset"`          // skillset name for this session
	Skill            string                 `json:"skill"`             // skill name to execute
	View             string                 `json:"view"`              // view name for policy evaluation
	ViewDefinition   *policy.ViewDefinition `json:"view_definition"`   // policy view definition
	SessionVariables map[string]any         `json:"session_variables"` // session-scoped variables
	InputArgs        map[string]any         `json:"input_args"`        // input arguments for skill execution
	Catalog          string                 `json:"catalog"`           // catalog name
	Variant          string                 `json:"variant"`           // variant name
	Namespace        string                 `json:"namespace"`         // namespace for resource isolation
	TenantID         catcommon.TenantId     `json:"tenant_id"`         // tenant identifier
}

var sessionManager *activeSessions

// CreateSession creates a new session with the given context and authentication token.
// Returns the created session and any error encountered during creation.
// SessionID must be valid and unique within the session manager.
func (as *activeSessions) CreateSession(ctx context.Context, c *ServerContext, token string, tokenExpiry time.Time, sessionType tangentcommon.SessionType) (*session, apperrors.Error) {
	if c.SessionID == uuid.Nil {
		return nil, ErrInvalidSession
	}
	// if a session with the same ID already exists, return an error
	if _, exists := as.sessions[c.SessionID]; exists {
		return nil, ErrAlreadyExists.New("session already exists")
	}
	session := &session{
		id:            c.SessionID,
		context:       c,
		skillSet:      nil,
		viewDef:       nil,
		token:         token,
		tokenExpiry:   tokenExpiry,
		callGraph:     toolgraph.NewCallGraph(3), // max depth of 3
		invocationIDs: make(map[string]*policy.ViewDefinition),
		sessionType:   sessionType,
	}
	logger := log.Ctx(ctx)
	if logger == nil {
		newLogger := log.With().Str("session_id", c.SessionID.String()).Logger()
		logger = &newLogger
	}
	session.logger = logger
	session.auditLogInfo.auditLogger = session.getLogger(TopicAuditLog)
	session.auditLogInfo.auditLogPubKey = config.GetRuntimeConfig().LogSigningKey.PublicKey
	as.sessions[c.SessionID] = session
	return session, nil
}

// GetSession retrieves a session by its unique identifier.
// Returns the session and any error encountered during retrieval.
func (as *activeSessions) GetSession(id uuid.UUID) (*session, apperrors.Error) {
	if session, exists := as.sessions[id]; exists {
		return session, nil
	}
	return nil, ErrInvalidSession
}

// ListSessions returns all active sessions in the session manager.
// Returns the session list and any error encountered during listing.
func (as *activeSessions) ListSessions() ([]*session, apperrors.Error) {
	var sessionList []*session
	for _, session := range as.sessions {
		sessionList = append(sessionList, session)
	}
	return sessionList, nil
}

// DeleteSession removes a session from the session manager.
// Cleans up associated event bus subscriptions and resources.
func (as *activeSessions) DeleteSession(id uuid.UUID) apperrors.Error {
	if _, exists := as.sessions[id]; !exists {
		return ErrInvalidSession
	}
	GetEventBus().CloseAllForPattern(GetAllSessionTopics(id.String()))
	delete(as.sessions, id)
	return nil
}

func init() {
	sessionManager = &activeSessions{
		sessions: make(map[uuid.UUID]*session),
	}
}

// ActiveSessionManager returns the global session manager instance.
// Provides access to session lifecycle management functions.
func ActiveSessionManager() SessionManager {
	return sessionManager
}
