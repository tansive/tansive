package tangentcommon

// SessionCreateRequest represents a request to create a new session.
// Contains authentication and session configuration parameters.
type SessionCreateRequest struct {
	Interactive  bool   `json:"interactive"`   // whether the session should be interactive
	CodeVerifier string `json:"code_verifier"` // PKCE code verifier for OAuth flow
	Code         string `json:"code"`          // authorization code for session creation
}
