package auth

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
)

func newDb() context.Context {
	config.TestInit()
	db.Init()
	ctx := log.Logger.WithContext(context.Background())
	ctx, err := db.ConnCtx(ctx)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("unable to get db connection")
	}
	ctx = catcommon.WithCatalogContext(ctx, &catcommon.CatalogContext{
		UserContext: &catcommon.UserContext{
			UserID: "user/test_user",
		},
	})
	return ctx
}

func replaceTabsWithSpaces(s *string) {
	*s = strings.ReplaceAll(*s, "\t", "    ")
}

var _ = replaceTabsWithSpaces
