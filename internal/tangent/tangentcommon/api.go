package tangentcommon

// SessionCreateRequest represents a request to create a new session.
// Contains authentication and session configuration parameters.
type SessionType string

const (
	SessionTypeInteractive    SessionType = "interactive"
	SessionTypeNonInteractive SessionType = "non-interactive"
	SessionTypeMCPProxy       SessionType = "mcp-proxy"
)

// This maintains code_verifier with _ instead of camel case to match the OAuth spec.
type SessionCreateRequest struct {
	SessionType  SessionType `json:"sessionType"`   // type of session to create
	CodeVerifier string      `json:"code_verifier"` // PKCE code verifier for OAuth flow
	Code         string      `json:"code"`          // authorization code for session creation
}
