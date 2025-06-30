package models

import (
	"time"

	"github.com/jackc/pgtype"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/uuid"
)

/*
   Column    |          Type           | Collation | Nullable |      Default
-------------+-------------------------+-----------+----------+--------------------
 catalog_id  | uuid                    |           | not null | uuid_generate_v4()
 name        | character varying(128)  |           | not null |
 description | character varying(1024) |           |          |
 info        | jsonb                   |           |          |
 project_id  | character varying(10)   |           | not null |
 tenant_id   | character varying(10)   |           | not null |
 created_at  | timestamptz            |           | not null | now()
 updated_at  | timestamptz            |           | not null | now()
*/

// Catalog model definition
type Catalog struct {
	CatalogID   uuid.UUID           `db:"catalog_id"`
	Name        string              `db:"name"`
	Description string              `db:"description"`
	Info        pgtype.JSONB        `db:"info"`
	ProjectID   catcommon.ProjectId `db:"project_id"`
	CreatedAt   time.Time           `db:"created_at"`
	UpdatedAt   time.Time           `db:"updated_at"`
}
