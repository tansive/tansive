package tangent

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tansive/tansive/internal/catalogsrv/apis"
	"github.com/tansive/tansive/internal/catalogsrv/auth"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/httpx"
)

var tangentHandlers = []policy.ResponseHandlerParam{
	{
		Method:  http.MethodPost,
		Path:    "/",
		Handler: createTangent,
	},
}

var tangentUserHandlers = []policy.ResponseHandlerParam{
	{
		Method: http.MethodGet,
		Path:   "/onboardingKey",
		//		Handler:        getOnboardingKey,
		AllowedActions: []policy.Action{policy.ActionTangentCreate},
	},
}

func Router() chi.Router {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.ContextMiddleware)
		r.Use(apis.CatalogContextLoader)
		for _, handler := range tangentUserHandlers {
			policyEnforcedHandler := policy.EnforceViewPolicyMiddleware(handler)
			r.Method(handler.Method, handler.Path, httpx.WrapHttpRsp(policyEnforcedHandler))
		}
	})
	r.Group(func(r chi.Router) {
		r.Use(tangentContextMiddleware)
		for _, handler := range tangentHandlers {
			r.Method(handler.Method, handler.Path, httpx.WrapHttpRsp(handler.Handler))
		}
	})
	return r
}

func tangentContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if config.Config().SingleUserMode {
			ctx = catcommon.WithTenantID(ctx, catcommon.TenantId(config.Config().DefaultTenantID))
			ctx = catcommon.WithCatalogContext(ctx, &catcommon.CatalogContext{
				UserContext: &catcommon.UserContext{
					UserID: "default-user",
				},
			})
		} else {
			httpx.ErrUnAuthorized("tangent is not supported in multi-user mode").Send(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
