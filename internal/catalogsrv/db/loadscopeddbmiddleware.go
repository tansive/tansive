package db

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/httpx"
)

// LoadScopedDBMiddleware is a middleware that loads a scoped db connection from the request context
// and closes it after the request is served.
func LoadScopedDBMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := ConnCtx(r.Context())
		if err != nil {
			log.Ctx(r.Context()).Fatal().Err(err).Msg("unable to get db connection")
			httpx.ErrApplicationError("unable to service request at this time").Send(w)
			return
		}
		defer func() {
			if dbConn := DB(ctx); dbConn != nil {
				dbConn.Close(context.Background()) // use background to avoid canceled context
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
