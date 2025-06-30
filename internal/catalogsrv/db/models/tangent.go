package models

import (
	"encoding/json"
	"time"

	"github.com/tansive/tansive/internal/common/uuid"
)

type Tangent struct {
	ID        uuid.UUID       `db:"id"`
	Info      json.RawMessage `db:"info"`
	PublicKey []byte          `db:"public_key"`
	Status    string          `db:"status"`
	TenantID  string          `db:"tenant_id"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}
