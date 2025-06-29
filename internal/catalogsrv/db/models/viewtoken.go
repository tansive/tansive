package models

import (
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/common/uuid"
)

type ViewToken struct {
	TokenID   uuid.UUID          `json:"token_id"`
	ViewID    uuid.UUID          `json:"view_id"`
	TenantID  catcommon.TenantId `json:"tenant_id"`
	ExpireAt  time.Time          `json:"expire_at"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

func (vt *ViewToken) Validate() error {
	if vt.ViewID == uuid.Nil {
		return dberror.ErrInvalidInput.Msg("view_id is required")
	}
	return nil
}
