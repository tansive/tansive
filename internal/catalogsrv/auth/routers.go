package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tansive/tansive/internal/catalogsrv/auth/userauth"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/httpx"
)

var authHandlers = []policy.ResponseHandlerParam{
	{
		Method:  http.MethodPost,
		Path:    "/view-adoptions/{catalogRef}/{viewLabel}",
		Handler: adoptView,
	},
	{
		Method:  http.MethodPost,
		Path:    "/default-view-adoptions/{catalogRef}",
		Handler: adoptDefaultCatalogView,
	},
}

// Router creates and configures a new router for authentication-related endpoints.
// It sets up middleware and registers handlers for various HTTP methods and paths.
func Router(r chi.Router) chi.Router {
	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Method(http.MethodPost, "/login", httpx.WrapHttpRsp(userauth.LoginUser))
	})
	router.Group(func(r chi.Router) {
		r.Use(UserAuthMiddleware)
		r.Use(LoadContext)
		for _, handler := range authHandlers {
			r.Method(handler.Method, handler.Path, httpx.WrapHttpRsp(handler.Handler))
		}
	})
	return router
}

func LoadContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Load projectID from URL query parameter
		projectID := r.URL.Query().Get("project")
		if projectID != "" {
			ctx = catcommon.WithProjectID(ctx, catcommon.ProjectId(projectID))
		} else if config.Config().SingleUserMode {
			ctx = catcommon.WithProjectID(ctx, catcommon.ProjectId(config.Config().DefaultProjectID))
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
