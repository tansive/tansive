package models

import (
	"encoding/json"
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/uuid"
)

type Session struct {
	SessionID     uuid.UUID          `db:"session_id"`
	SkillSet      string             `db:"skillset"`
	Skill         string             `db:"skill"`
	ViewID        uuid.UUID          `db:"view_id"`
	TangentID     uuid.UUID          `db:"tangent_id"`
	StatusSummary string             `db:"status_summary"`
	Status        json.RawMessage    `db:"status"`
	Info          json.RawMessage    `db:"info"`
	UserID        string             `db:"user_id"`
	CatalogID     uuid.UUID          `db:"catalog_id"`
	VariantID     uuid.UUID          `db:"variant_id"`
	TenantID      catcommon.TenantId `db:"tenant_id"`
	CreatedAt     time.Time          `db:"created_at"`
	StartedAt     time.Time          `db:"started_at"`
	EndedAt       time.Time          `db:"ended_at"`
	UpdatedAt     time.Time          `db:"updated_at"`
	ExpiresAt     time.Time          `db:"expires_at"`
}
