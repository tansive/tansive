package apis

import (
	"net/http"
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/httpx"
)

func getObject(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	var kind string

	reqContext, err := hydrateRequestContext(r)
	if err != nil {
		return nil, err
	}

	kind = getResourceKind(r)
	if kind == catcommon.InvalidKind {
		return nil, httpx.ErrInvalidRequest()
	}

	rm, err := catalogmanager.ResourceManagerForKind(ctx, kind, reqContext)
	if err != nil {
		return nil, err
	}

	rsrc, err := rm.Get(ctx)
	if err != nil {
		return nil, err
	}

	rsp := &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   rsrc,
	}
	return rsp, nil
}

type StatusRsp struct {
	UserID        string                 `json:"userID,omitempty"`
	ServerTime    string                 `json:"serverTime,omitempty"`
	ServerVersion string                 `json:"serverVersion"`
	ApiVersion    string                 `json:"apiVersion"`
	ViewDef       *policy.ViewDefinition `json:"viewDef,omitempty"`
}

func getStatus(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	viewDef, _ := policy.ResolveAuthorizedViewDef(ctx)
	serverVersion := catcommon.ServerVersion
	apiVersion := catcommon.ApiVersion
	serverTime := time.Now().Format(time.RFC3339)
	var userID string
	userContext := catcommon.GetUserContext(ctx)
	if userContext != nil {
		userID = userContext.UserID
	}

	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response: &StatusRsp{
			UserID:        userID,
			ServerTime:    serverTime,
			ServerVersion: serverVersion,
			ApiVersion:    apiVersion,
			ViewDef:       viewDef,
		},
	}, nil
}
