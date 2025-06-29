package test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"encoding/json"

	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	catalogsrv "github.com/tansive/tansive/internal/catalogsrv/server"
	"github.com/tansive/tansive/internal/catalogsrv/session"
)

type TestContext struct {
	TenantId  catcommon.TenantId
	ProjectId catcommon.ProjectId
	catcommon.CatalogContext
}

func ExecuteTestRequest(t *testing.T, req *http.Request, apiKey *string, testContext ...TestContext) *httptest.ResponseRecorder {
	s, err := catalogsrv.CreateNewServer()
	assert.NoError(t, err, "create new server")

	if apiKey != nil {
		_ = apiKey
		//auth.SignApiRequest(req, apiKey.KeyId, apiKey.PrivKey)
	}

	// Mount Handlers
	s.MountHandlers()

	rr := httptest.NewRecorder()
	if len(testContext) > 0 {
		ctx := req.Context()
		ctx = catcommon.WithTenantID(ctx, testContext[0].TenantId)
		ctx = catcommon.WithProjectID(ctx, testContext[0].ProjectId)
		catalogContext := &testContext[0].CatalogContext
		vd := policy.ViewDefinition{
			Scope: policy.Scope{
				Catalog: testContext[0].CatalogContext.Catalog,
			},
			Rules: []policy.Rule{
				{
					Intent:  policy.IntentAllow,
					Actions: []policy.Action{policy.ActionCatalogAdmin},
					Targets: []policy.TargetResource{},
				},
			},
		}
		u := catcommon.UserContext{
			UserID: "user/test_user",
		}
		catalogContext.UserContext = &u
		ctx = policy.WithViewDefinition(ctx, &vd)
		ctx = catcommon.WithCatalogContext(ctx, catalogContext)
		ctx = catcommon.WithTestContext(ctx, true)
		req = req.WithContext(ctx)
	}
	s.Router.ServeHTTP(rr, req)

	return rr
}

func CheckHeader(t *testing.T, h http.Header) {
	expected := "application/json"
	got := h.Get("Content-Type")
	assert.Equal(t, expected, got, "Content-Type expected %s, got %s", expected, got)
	assert.NotNil(t, h.Get("X-Request-ID"), "No Request Id")
}

func CompareJson(t *testing.T, expected any, actual string) {
	j, err := json.Marshal(expected)
	assert.NoError(t, err, "json marshal")
	assert.JSONEq(t, string(j), actual, "Expected: %v\n Got: %v\n", expected, actual)
}

var _ = SetRequestBodyAndHeader

func SetRequestBodyAndHeader(t *testing.T, req *http.Request, data interface{}) {
	// Marshal the data into JSON
	// check if the input itsef is json
	var jsonData []byte
	if s, ok := data.(string); ok {
		if json.Valid([]byte(s)) {
			jsonData = []byte(s)
		}
	} else if b, ok := data.([]byte); ok {
		if json.Valid(b) {
			jsonData = b
		}
	} else {
		var err error
		jsonData, err = json.Marshal(data)
		assert.NoError(t, err, "Failed to marshal data into JSON")
	}

	// Set the request body to the JSON
	req.Body = io.NopCloser(bytes.NewReader(jsonData))
	req.ContentLength = int64(len(jsonData))

	// Set the Content-Type header to application/json
	req.Header.Set("Content-Type", "application/json")
}

func NewDb(t *testing.T) context.Context {
	config.TestInit()
	db.Init()
	session.Init()
	ctx := log.Logger.WithContext(context.Background())
	ctx, err := db.ConnCtx(ctx)
	t.Cleanup(func() {
		db.DB(ctx).Close(ctx)
		log.Ctx(ctx).Info().Msg("closed db connection")
	})
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("unable to get db connection")
	}
	return ctx
}
