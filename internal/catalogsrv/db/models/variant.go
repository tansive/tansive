package models

import (
	"time"

	"github.com/jackc/pgtype"
	"github.com/tansive/tansive/internal/common/uuid"
)

/*
 Table "public.variants"
          Column          |           Type           | Collation | Nullable |      Default
--------------------------+--------------------------+-----------+----------+--------------------
 variant_id               | uuid                     |           | not null | uuid_generate_v4()
 name                     | character varying(128)   |           | not null |
 description              | character varying(1024)  |           |          |
 info                     | jsonb                    |           |          |
 resource_directory       | uuid                     |           |          | uuid_nil()
 skillset_directory       | uuid                     |           |          | uuid_nil()
 catalog_id               | uuid                     |           | not null |
 tenant_id                | character varying(10)    |           | not null |
 created_at               | timestamp with time zone |           |          | now()
 updated_at               | timestamp with time zone |           |          | now()
Indexes:
    "variants_pkey" PRIMARY KEY, btree (variant_id, tenant_id)
    "variants_name_catalog_id_tenant_id_key" UNIQUE CONSTRAINT, btree (name, catalog_id, tenant_id)
Check constraints:
    "variants_name_check" CHECK (name::text ~ '^[A-Za-z0-9_-]+$'::text)
Foreign-key constraints:
    "variants_catalog_id_tenant_id_fkey" FOREIGN KEY (catalog_id, tenant_id) REFERENCES catalogs(catalog_id, tenant_id) ON DELETE CASCADE
Referenced by:
    TABLE "namespaces" CONSTRAINT "namespaces_variant_id_tenant_id_fkey" FOREIGN KEY (variant_id, tenant_id) REFERENCES variants(variant_id, tenant_id) ON DELETE CASCADE
    TABLE "resourcegroups_directory" CONSTRAINT "resourcegroups_directory_variant_id_tenant_id_fkey" FOREIGN KEY (variant_id, tenant_id) REFERENCES variants(variant_id, tenant_id) ON DELETE CASCADE
Triggers:
    update_variants_updated_at BEFORE UPDATE ON variants FOR EACH ROW EXECUTE FUNCTION set_updated_at()
*/

// Variant model definition
type Variant struct {
	VariantID           uuid.UUID    `db:"variant_id"`
	Name                string       `db:"name"`
	Description         string       `db:"description"`
	Info                pgtype.JSONB `db:"info"`
	CatalogID           uuid.UUID    `db:"catalog_id"`
	ResourceDirectoryID uuid.UUID    `db:"resource_directory"`
	SkillsetDirectoryID uuid.UUID    `db:"skillset_directory"`
	CreatedAt           time.Time    `db:"created_at"`
	UpdatedAt           time.Time    `db:"updated_at"`
}

// VariantSummary represents a simplified variant with just name, ID, and directory IDs
type VariantSummary struct {
	VariantID           uuid.UUID `db:"variant_id"`
	Name                string    `db:"name"`
	ResourceDirectoryID uuid.UUID `db:"resource_directory"`
	SkillsetDirectoryID uuid.UUID `db:"skillset_directory"`
}
