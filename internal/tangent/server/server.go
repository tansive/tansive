// Package server provides HTTP server functionality for the Tangent runtime.
// It implements a RESTful API server with session management, version information,
// and health check endpoints. The package supports CORS handling and middleware
// integration for logging and error handling.
package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"

	"github.com/tansive/tansive/internal/common/httpx"
	"github.com/tansive/tansive/internal/common/logtrace"
	"github.com/tansive/tansive/internal/common/middleware"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/session"
)

// AgentServer provides the main HTTP server for the Tangent runtime.
// Manages routing, middleware, and endpoint handling for session operations.
type AgentServer struct {
	Router *chi.Mux // HTTP router for request handling
}

// CreateNewServer creates a new AgentServer instance.
// Returns the server instance and any error encountered during creation.
func CreateNewServer() (*AgentServer, error) {
	s := &AgentServer{}
	s.Router = chi.NewRouter()
	return s, nil
}

// MountHandlers sets up all HTTP routes and middleware for the server.
// Configures logging, panic handling, CORS, and resource endpoints.
func (s *AgentServer) MountHandlers() {
	s.Router.Use(middleware.RequestLogger)
	s.Router.Use(middleware.PanicHandler)
	if config.Config().HandleCORS {
		s.Router.Use(s.HandleCORS)
	}
	s.mountResourceHandlers(s.Router)
	if logtrace.IsTraceEnabled() {
		fmt.Println("Routes in tangent router")
		walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			fmt.Printf("%s %s\n", method, route)
			return nil
		}
		if err := chi.Walk(s.Router, walkFunc); err != nil {
			log.Error().Err(err).Msg("Error walking router")
		}
	}
}

// mountResourceHandlers registers all resource endpoints on the router.
// Sets up session management routes and system endpoints.
func (s *AgentServer) mountResourceHandlers(r chi.Router) {
	r.Route("/sessions", func(r chi.Router) {
		session.Router(r)
	})
	r.Get("/version", s.getVersion)
	r.Get("/ready", s.getReadiness)
}

// GetVersionRsp represents the response for version information.
// Contains server and API version details.
type GetVersionRsp struct {
	ServerVersion string `json:"serverVersion"` // server version string
	ApiVersion    string `json:"apiVersion"`    // API version string
}

// getVersion handles version information requests.
// Returns server and API version information in JSON format.
func (s *AgentServer) getVersion(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Debug().Msg("GetVersion")
	rsp := &GetVersionRsp{
		ServerVersion: "Tansive Tangent Server: " + Version,
		ApiVersion:    session.Version,
	}
	httpx.SendJsonRsp(r.Context(), w, http.StatusOK, rsp)
}

// getReadiness handles health check requests.
// Returns readiness status for load balancer and monitoring systems.
func (s *AgentServer) getReadiness(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Debug().Msg("Readiness check")

	// Add any specific readiness checks here
	// For now, we'll just return ready if the server is up
	httpx.SendJsonRsp(r.Context(), w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

// HandleCORS provides CORS middleware for cross-origin requests.
// Configures allowed origins, methods, headers, and credentials handling.
func (s *AgentServer) HandleCORS(next http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, //TODO: Change this to specific origin
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Content-Length", "Accept-Encoding"},
		ExposedHeaders:   []string{"Link", "Location", "X-Tansive-Request-Id"},
		AllowCredentials: false,
		MaxAge:           300,
	})(next)
}
