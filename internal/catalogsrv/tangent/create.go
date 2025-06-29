package tangent

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/httpx"
	"github.com/tansive/tansive/internal/common/uuid"
)

func createTangent(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()

	if r.Body == nil {
		return nil, httpx.ErrInvalidRequest("request body is required")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, httpx.ErrUnableToReadRequest()
	}

	req := &TangentInfo{}
	if err := json.Unmarshal(body, req); err != nil {
		return nil, httpx.ErrInvalidRequest("invalid request body")
	}

	if req.ID == uuid.Nil {
		return nil, httpx.ErrInvalidRequest("id is required")
	}

	info, err := json.Marshal(req)
	if err != nil {
		return nil, httpx.ErrInvalidRequest("invalid request body")
	}
	t := models.Tangent{
		ID:        req.ID,
		PublicKey: req.PublicKeyAccessKey,
		Info:      info,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.DB(ctx).CreateTangent(ctx, &t); err != nil {
		if errors.Is(err, dberror.ErrAlreadyExists) {
			log.Ctx(ctx).Error().Err(err).Msg("tangent already exists, updating")
			if err := db.DB(ctx).UpdateTangent(ctx, &t); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to update tangent")
				return nil, err
			} else {
				return &httpx.Response{
					StatusCode: http.StatusCreated,
					Location:   "/tangents/" + req.ID.String(),
					Response:   nil,
				}, nil
			}
		}
		return nil, err
	}

	return &httpx.Response{
		StatusCode: http.StatusCreated,
		Location:   "/tangents/" + req.ID.String(),
		Response:   nil,
	}, nil
}
