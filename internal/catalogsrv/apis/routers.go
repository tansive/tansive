package apis

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/auth"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/httpx"
)

var userSessionHandlers = []policy.ResponseHandlerParam{
	{
		Method:  http.MethodGet,
		Path:    "/views",
		Handler: listObjects,
	},
	{
		Method:  http.MethodPost,
		Path:    "/catalogs",
		Handler: createObject,
	},
	{
		Method:  http.MethodGet,
		Path:    "/catalogs",
		Handler: listObjects,
	},
}

// resourceObjectHandlers defines the API routes and their authorization requirements.
// Each route requires at least one of the listed actions to be authorized.
var resourceObjectHandlers = []policy.ResponseHandlerParam{
	{
		Method:         http.MethodGet,
		Path:           "/catalogs/{catalogName}",
		Handler:        getObject,
		AllowedActions: []policy.Action{policy.ActionCatalogList},
	},
	{
		Method:         http.MethodPut,
		Path:           "/catalogs/{catalogName}",
		Handler:        updateObject,
		AllowedActions: []policy.Action{policy.ActionCatalogAdmin},
	},
	{
		Method:         http.MethodDelete,
		Path:           "/catalogs/{catalogName}",
		Handler:        deleteObject,
		AllowedActions: []policy.Action{policy.ActionCatalogAdmin},
	},
	{
		Method:         http.MethodPost,
		Path:           "/variants",
		Handler:        createObject,
		AllowedActions: []policy.Action{policy.ActionVariantClone},
	},
	{
		Method:         http.MethodGet,
		Path:           "/variants/{variantName}",
		Handler:        getObject,
		AllowedActions: []policy.Action{policy.ActionVariantList},
	},
	{
		Method:         http.MethodPut,
		Path:           "/variants/{variantName}",
		Handler:        updateObject,
		AllowedActions: []policy.Action{policy.ActionVariantAdmin},
	},
	{
		Method:         http.MethodDelete,
		Path:           "/variants/{variantName}",
		Handler:        deleteObject,
		AllowedActions: []policy.Action{policy.ActionVariantAdmin},
	},
	{
		Method:         http.MethodPost,
		Path:           "/namespaces",
		Handler:        createObject,
		AllowedActions: []policy.Action{policy.ActionNamespaceCreate},
	},
	{
		Method:         http.MethodGet,
		Path:           "/namespaces/{namespaceName}",
		Handler:        getObject,
		AllowedActions: []policy.Action{policy.ActionNamespaceList},
	},
	{
		Method:         http.MethodPut,
		Path:           "/namespaces/{namespaceName}",
		Handler:        updateObject,
		AllowedActions: []policy.Action{policy.ActionNamespaceAdmin},
	},
	{
		Method:         http.MethodDelete,
		Path:           "/namespaces/{namespaceName}",
		Handler:        deleteObject,
		AllowedActions: []policy.Action{policy.ActionNamespaceAdmin},
	},
	{
		Method:         http.MethodPost,
		Path:           "/views",
		Handler:        createObject,
		AllowedActions: []policy.Action{policy.ActionCatalogCreateView},
	},
	{
		Method:         http.MethodGet,
		Path:           "/status",
		Handler:        getStatus,
		AllowedActions: []policy.Action{policy.ActionAllow},
		Options:        []policy.HandlerOptions{policy.SkipViewDefValidation(true)},
	},
	{
		Method:         http.MethodGet,
		Path:           "/views/{viewName}",
		Handler:        getObject,
		AllowedActions: []policy.Action{policy.ActionCatalogList},
	},
	{
		Method:         http.MethodPut,
		Path:           "/views/{viewName}",
		Handler:        updateObject,
		AllowedActions: []policy.Action{policy.ActionViewAdmin},
	},
	{
		Method:         http.MethodDelete,
		Path:           "/views/{viewName}",
		Handler:        deleteObject,
		AllowedActions: []policy.Action{policy.ActionViewAdmin},
	},
	{
		Method:         http.MethodPost,
		Path:           "/resources",
		Handler:        createObject,
		AllowedActions: []policy.Action{policy.ActionResourceCreate},
	},
	{
		Method:         http.MethodGet,
		Path:           "/resources",
		Handler:        listObjects,
		AllowedActions: []policy.Action{policy.ActionResourceList},
	},
	{
		Method:         http.MethodGet,
		Path:           "/resources/definition/*",
		Handler:        getObject,
		AllowedActions: []policy.Action{policy.ActionResourceRead, policy.ActionResourceEdit},
	},
	{
		Method:         http.MethodPut,
		Path:           "/resources/definition/*",
		Handler:        updateObject,
		AllowedActions: []policy.Action{policy.ActionResourceEdit},
	},
	{
		Method:         http.MethodDelete,
		Path:           "/resources/definition/*",
		Handler:        deleteObject,
		AllowedActions: []policy.Action{policy.ActionResourceDelete},
	},
	{
		Method:         http.MethodGet,
		Path:           "/resources/*",
		Handler:        getObject,
		AllowedActions: []policy.Action{policy.ActionResourceGet, policy.ActionResourcePut},
	},
	{
		Method:         http.MethodPut,
		Path:           "/resources/*",
		Handler:        updateObject,
		AllowedActions: []policy.Action{policy.ActionResourcePut},
	},
	{
		Method:         http.MethodPost,
		Path:           "/skillsets",
		Handler:        createObject,
		AllowedActions: []policy.Action{policy.ActionSkillSetCreate},
	},
	{
		Method:         http.MethodGet,
		Path:           "/skillsets",
		Handler:        listObjects,
		AllowedActions: []policy.Action{policy.ActionSkillSetList},
	},
	{
		Method:         http.MethodGet,
		Path:           "/skillsets/*",
		Handler:        getObject,
		AllowedActions: []policy.Action{policy.ActionSkillSetRead, policy.ActionSkillSetUse},
	},
	{
		Method:         http.MethodPut,
		Path:           "/skillsets/*",
		Handler:        updateObject,
		AllowedActions: []policy.Action{policy.ActionSkillSetAdmin},
	},
	{
		Method:         http.MethodDelete,
		Path:           "/skillsets/*",
		Handler:        deleteObject,
		AllowedActions: []policy.Action{policy.ActionSkillSetAdmin},
	},
}

// Router creates and configures a new router for catalog service API endpoints.
// It sets up middleware and registers handlers for various HTTP methods and paths.
func Router(r chi.Router) chi.Router {
	//router := chi.NewRouter()
	//Load the group that needs only user session/identity validation
	r.Group(func(r chi.Router) {
		r.Use(auth.UserAuthMiddleware)
		r.Use(CatalogContextLoader)
		for _, handler := range userSessionHandlers {
			r.Method(handler.Method, handler.Path, httpx.WrapHttpRsp(handler.Handler))
		}
	})

	//Load the group that needs session validation and catalog context
	r.Group(func(r chi.Router) {
		r.Use(auth.ContextMiddleware)
		r.Use(CatalogContextLoader)
		for _, handler := range resourceObjectHandlers {
			//Wrap the request handler with view policy enforcement
			policyEnforcedHandler := policy.EnforceViewPolicyMiddleware(handler)
			r.Method(handler.Method, handler.Path, httpx.WrapHttpRsp(policyEnforcedHandler))
		}
	})
	return r
}

// CatalogContextLoader is a middleware that loads and validates catalog context information
// from the request context and URL parameters. It ensures that tenant and project IDs
// are present and loads related objects (catalog, variant, workspace, namespace).
func CatalogContextLoader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tenantID := catcommon.GetTenantID(ctx)

		if tenantID == "" {
			httpx.ErrInvalidRequest().Send(w)
			return
		}

		c := catcommon.GetCatalogContext(ctx)
		if c == nil {
			httpx.ErrUnAuthorized("missing or invalid authorization token").Send(w)
			return
		}

		r, err := withContext(r)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				log.Ctx(ctx).Error().Msgf("request body too large (limit: %d bytes)", maxErr.Limit)
				httpx.ErrRequestTooLarge(maxErr.Limit).Send(w)
			} else {
				httpx.ErrInvalidRequest(err.Error()).Send(w)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}
