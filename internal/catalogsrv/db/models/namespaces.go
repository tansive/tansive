package models

import (
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/uuid"
)

/*
Column    |           Type           | Collation | Nullable | Default
-------------+--------------------------+-----------+----------+---------
 name        | character varying(128)   |           | not null |
 variant_id  | uuid                     |           | not null |
 tenant_id   | character varying(10)    |           | not null |
 description | character varying(1024)  |           |          |
 info        | jsonb                    |           |          |
 created_at  | timestamp with time zone |           |          | now()
 updated_at  | timestamp with time zone |           |          | now()
Indexes:
    "namespaces_pkey" PRIMARY KEY, btree (name, variant_id, tenant_id)
Check constraints:
    "namespaces_name_check" CHECK (name::text ~ '^[A-Za-z0-9_-]+$'::text)
Foreign-key constraints:
    "namespaces_tenant_id_fkey" FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
    "namespaces_variant_id_tenant_id_fkey" FOREIGN KEY (variant_id, tenant_id) REFERENCES variants(variant_id, tenant_id) ON DELETE CASCADE
*/

type Namespace struct {
	Name        string             `db:"name"`
	VariantID   uuid.UUID          `db:"variant_id"`
	TenantID    catcommon.TenantId `db:"tenant_id"`
	Description string             `db:"description"`
	Info        []byte             `db:"info"`
	CatalogID   uuid.UUID          `db:"-"`
	Catalog     string             `db:"-"`
	Variant     string             `db:"-"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
}
