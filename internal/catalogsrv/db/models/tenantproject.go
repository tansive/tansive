package models

import (
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
)

type Tenant struct {
	TenantID  catcommon.TenantId
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Project struct {
	ProjectID catcommon.ProjectId
	TenantID  catcommon.TenantId
	CreatedAt time.Time
	UpdatedAt time.Time
}
