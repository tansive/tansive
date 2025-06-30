package session

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tansive/tansive/internal/common/httpx"
)

// ResponseHandlerParam defines the configuration for HTTP route handlers.
// Contains HTTP method, path, and handler function for route registration.
type ResponseHandlerParam struct {
	Method  string               // HTTP method (GET, POST, etc.)
	Path    string               // URL path pattern
	Handler httpx.RequestHandler // handler function for the route
}

var resourceObjectHandlers = []ResponseHandlerParam{
	{
		Method:  http.MethodPost,
		Path:    "/",
		Handler: createSession,
	},
}

// Router sets up HTTP routes for session management.
// Registers session creation endpoint and applies authentication middleware.
func Router(r chi.Router) {
	r.Use(SessionAuthenticator)
	for _, handler := range resourceObjectHandlers {
		r.Method(handler.Method, handler.Path, httpx.WrapHttpRsp(handler.Handler))
	}
	// Route for connection that'll upgrade HTTP to WebSocket
	//	r.Method(http.MethodGet, "/{id}/channel", http.HandlerFunc(getSessionChannel))
}

// SessionAuthenticator provides authentication middleware for session endpoints.
// Currently a placeholder for future authentication implementation.
func SessionAuthenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// implement authentication logic here
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
