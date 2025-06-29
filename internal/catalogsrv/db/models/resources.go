package models

import (
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/uuid"
)

type Resource struct {
	ID        uuid.UUID          `db:"id"`
	Path      string             `db:"path"`
	Hash      string             `db:"hash"`
	VariantID uuid.UUID          `db:"variant_id"`
	TenantID  catcommon.TenantId `db:"tenant_id"`
	CreatedAt time.Time          `db:"created_at"`
	UpdatedAt time.Time          `db:"updated_at"`
}
