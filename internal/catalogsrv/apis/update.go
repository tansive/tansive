package apis

import (
	"io"
	"net/http"

	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/httpx"
)

// Create a new resource object
func updateObject(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	var kind string

	if r.Body == nil {
		return nil, httpx.ErrInvalidRequest()
	}
	req, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, httpx.ErrUnableToReadRequest()
	}

	reqContext, err := hydrateRequestContext(r)
	if err != nil {
		return nil, err
	}
	kind = getResourceKind(r)
	if kind == catcommon.InvalidKind {
		return nil, httpx.ErrInvalidRequest()
	}

	if err := validateRequest(req, kind); err != nil {
		return nil, err
	}

	rm, err := catalogmanager.ResourceManagerForKind(ctx, kind, reqContext)
	if err != nil {
		return nil, err
	}
	err = rm.Update(ctx, req)
	if err != nil {
		return nil, err
	}

	rsp := &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   nil,
	}
	return rsp, nil
}
