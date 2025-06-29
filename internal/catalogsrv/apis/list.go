package apis

import (
	"net/http"

	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/httpx"
)

func listObjects(r *http.Request) (*httpx.Response, error) {
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

	rsrc, err := rm.List(ctx)
	if err != nil {
		return nil, err
	}

	rsp := &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   rsrc,
	}
	return rsp, nil
}
