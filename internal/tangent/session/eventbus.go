package session

import (
	"fmt"

	"github.com/tansive/tansive/internal/tangent/eventbus"
)

var eventBus *eventbus.EventBus

const (
	// TopicInteractiveLog is the event topic for interactive session logs.
	// Used for streaming interactive session output and events.
	TopicInteractiveLog = "interactive.log"

	// TopicAuditLog is the event topic for audit logging events.
	// Used for security and compliance audit trail events.
	TopicAuditLog = "audit.log"

	// TopicSessionLog is the event topic for general session logs.
	// Used for session lifecycle and general logging events.
	TopicSessionLog = "session.log"
)

func init() {
	eventBus = eventbus.New()
	if eventBus == nil {
		panic("eventBus is nil")
	}
}

// GetEventBus returns the global event bus instance.
// Provides access to event publishing and subscription functionality.
func GetEventBus() *eventbus.EventBus {
	return eventBus
}

// GetAllSessionTopics generates a pattern for all topics related to a session.
// Returns a wildcard pattern that matches all topics for the given session ID.
func GetAllSessionTopics(sessionID string) string {
	return fmt.Sprintf("session.%s.*", sessionID)
}

// GetSessionTopic generates a topic name for a specific session and event type.
// Returns a formatted topic string for session-specific event routing.
func GetSessionTopic(sessionID string, topic string) string {
	return fmt.Sprintf("session.%s.%s", sessionID, topic)
}
