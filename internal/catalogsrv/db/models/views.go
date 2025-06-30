package models

import (
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/uuid"
)

/*
Column    |           Type           | Collation | Nullable | Default
-------------+--------------------------+-----------+----------+---------
 view_id     | uuid                     |           | not null | uuid_generate_v4()
 label       | character varying(128)   |           |          |
 description | character varying(1024)  |           |          |
 info        | jsonb                    |           |          |
 rules       | jsonb                    |           | not null |
 catalog_id  | uuid                     |           | not null |
 tenant_id   | character varying(10)    |           | not null |
 created_at  | timestamp with time zone |           |          | now()
 updated_at  | timestamp with time zone |           |          | now()
Indexes:
    "views_pkey" PRIMARY KEY, btree (view_id, tenant_id)
    "views_label_catalog_id_tenant_id_key" UNIQUE, btree (label, catalog_id, tenant_id)
Check constraints:
    "views_label_check" CHECK (label::text ~ '^[A-Za-z0-9_-]+$'::text)
Foreign-key constraints:
    "views_tenant_id_fkey" FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
    "views_catalog_id_tenant_id_fkey" FOREIGN KEY (catalog_id, tenant_id) REFERENCES catalogs(catalog_id, tenant_id) ON DELETE CASCADE
*/

type View struct {
	ViewID      uuid.UUID          `db:"view_id"`
	Label       string             `db:"label"`
	Description string             `db:"description"`
	Info        []byte             `db:"info"`
	Rules       []byte             `db:"rules"`
	CatalogID   uuid.UUID          `db:"catalog_id"`
	TenantID    catcommon.TenantId `db:"tenant_id"`
	Catalog     string             `db:"-"`
	CreatedBy   string             `db:"created_by"`
	UpdatedBy   string             `db:"updated_by"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
}
