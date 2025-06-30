package models

import (
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
)

/*
     Column   |         Type          | Collation | Nullable | Default
--------------+-----------------------+-----------+----------+---------
	hash      | character(128)        |           | not null |
	type      | character varying(64) |           | not null |
	tenant_id | character varying(10) |           | not null |
	data      | bytea                 |           | not null |
	created_at| timestamptz          |           | not null | now()
	updated_at| timestamptz          |           | not null | now()
*/

type CatalogObject struct {
	ID        int64                       `db:"id"`
	HashID    string                      `db:"hash_id"`
	Hash      string                      `db:"hash"`
	Type      catcommon.CatalogObjectType `db:"type"`
	Version   string                      `db:"version"`
	TenantID  catcommon.TenantId          `db:"tenant_id"`
	Data      []byte                      `db:"data"`
	CreatedAt time.Time                   `db:"created_at"`
	UpdatedAt time.Time                   `db:"updated_at"`
}
