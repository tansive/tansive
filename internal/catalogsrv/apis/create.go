package apis

import (
	"errors"
	"io"
	"net/http"

	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/httpx"
)

// createObject creates a new resource object
func createObject(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()

	if r.Body == nil {
		return nil, httpx.ErrInvalidRequest("request body is required")
	}

	req, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, httpx.ErrUnableToReadRequest()
	}

	reqContext, err := hydrateRequestContext(r)
	if err != nil {
		return nil, err
	}

	kind := getResourceKind(r)
	if kind == catcommon.InvalidKind {
		return nil, httpx.ErrInvalidRequest("invalid resource kind")
	}

	if err := validateRequest(req, kind); err != nil {
		return nil, err
	}

	manager, err := catalogmanager.ResourceManagerForKind(ctx, kind, reqContext)
	if err != nil {
		return nil, err
	}

	resourceLoc, err := manager.Create(ctx, req)
	if err != nil {
		if errors.Is(err, catalogmanager.ErrInvalidVariant) {
			return nil, httpx.ErrInvalidVariant()
		}
		return nil, err
	}

	resp := &httpx.Response{
		StatusCode: http.StatusCreated,
		Location:   resourceLoc,
		Response:   nil,
	}

	return resp, nil
}
