package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/apis"
	"github.com/tansive/tansive/internal/catalogsrv/auth"
	"github.com/tansive/tansive/internal/catalogsrv/auth/keymanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/session"
	"github.com/tansive/tansive/internal/catalogsrv/tangent"
	"github.com/tansive/tansive/internal/common/httpx"
	"github.com/tansive/tansive/internal/common/logtrace"
	commonmiddleware "github.com/tansive/tansive/internal/common/middleware"
)

type CatalogServer struct {
	Router *chi.Mux
	km     keymanager.KeyManager
}

func CreateNewServer() (*CatalogServer, error) {
	s := &CatalogServer{}
	s.Router = chi.NewRouter()

	// Use the singleton key manager instance
	s.km = keymanager.GetKeyManager()

	return s, nil
}

func (s *CatalogServer) MountHandlers() {
	s.Router.Use(commonmiddleware.RequestLogger)
	s.Router.Use(commonmiddleware.PanicHandler)
	s.Router.Use(db.LoadScopedDBMiddleware)
	if config.Config().HandleCORS {
		s.Router.Use(s.HandleCORS)
	}
	//s.Router.Route("/", s.mountResourceHandlers)
	s.mountResourceHandlers(s.Router)
	if logtrace.IsTraceEnabled() {
		//print all the routes in the router by transversing the tree and printing the patterns
		fmt.Println("Routes in tenant router")
		walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			fmt.Printf("%s %s\n", method, route)
			return nil
		}
		if err := chi.Walk(s.Router, walkFunc); err != nil {
			fmt.Printf("Logging err: %s\n", err.Error())
		}
	}
}

func (s *CatalogServer) mountResourceHandlers(r chi.Router) {
	apis.Router(r)
	r.Mount("/auth", auth.Router(r))
	r.Mount("/sessions", session.Router())
	r.Mount("/tangents", tangent.Router())
	r.Get("/version", s.getVersion)
	r.Get("/ready", s.getReadiness)
	r.Get("/.well-known/jwks.json", auth.GetJWKSHandler(s.km))
}

type GetVersionRsp struct {
	ServerVersion string `json:"serverVersion"`
	ApiVersion    string `json:"apiVersion"`
}

func (s *CatalogServer) getVersion(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Debug().Msg("GetVersion")
	rsp := &GetVersionRsp{
		ServerVersion: "Tansive Catalog Server: " + catcommon.ServerVersion,
		ApiVersion:    catcommon.ApiVersion,
	}
	httpx.SendJsonRsp(r.Context(), w, http.StatusOK, rsp)
}

func (s *CatalogServer) getReadiness(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Debug().Msg("Readiness check")

	// Check if we can get a database connection
	ctx, err := db.ConnCtx(r.Context())
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("Database connection failed during readiness check")
		httpx.SendJsonRsp(r.Context(), w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"error":  "database connection failed",
		})
		return
	}
	defer db.DB(ctx).Close(ctx)

	// If we get here, the server is ready
	httpx.SendJsonRsp(r.Context(), w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func (s *CatalogServer) HandleCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "http://local.tansive.dev:8190")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")                                                       // Allowed methods
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Hatch-IDToken") // Allowed headers

		// Check if the request method is OPTIONS
		if r.Method == "OPTIONS" {
			log.Ctx(r.Context()).Debug().Msg("OPTIONS request")
			// Respond with appropriate headers and no body
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
