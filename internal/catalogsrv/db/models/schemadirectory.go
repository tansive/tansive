package models

import (
	"time"

	"encoding/json"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/uuid"
)

/*
                       Table "public.resourcegroups_directory"
    Column    |           Type           | Collation | Nullable |      Default
--------------+--------------------------+-----------+----------+--------------------
 directory_id | uuid                     |           | not null | uuid_generate_v4()
 variant_id   | uuid                     |           | not null |
 tenant_id    | character varying(10)    |           | not null |
 directory    | jsonb                    |           | not null |
 created_at   | timestamp with time zone |           |          | now()
 updated_at   | timestamp with time zone |           |          | now()
Indexes:
    "resourcegroups_directory_pkey" PRIMARY KEY, btree (directory_id, tenant_id)
    "idx_resourcegroups_directory_hash_gin" gin (jsonb_path_query_array(directory, '$.*."hash"'::jsonpath))
Foreign-key constraints:
    "resourcegroups_directory_tenant_id_fkey" FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
    "resourcegroups_directory_variant_id_tenant_id_fkey" FOREIGN KEY (variant_id, tenant_id) REFERENCES variants(variant_id, tenant_id) ON DELETE CASCADE
Triggers:
    update_resourcegroups_directory_updated_at BEFORE UPDATE ON resourcegroups_directory FOR EACH ROW EXECUTE FUNCTION set_updated_at()

*/

type SchemaDirectory struct {
	DirectoryID uuid.UUID          `db:"directory_id"`
	VariantID   uuid.UUID          `db:"variant_id"`
	TenantID    catcommon.TenantId `db:"tenant_id"`
	Directory   []byte             `db:"directory"` // JSONB
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
}

type DirectoryID struct {
	ID   uuid.UUID
	Type catcommon.CatalogObjectType
}

type DirectoryIDs []DirectoryID

type ObjectRef struct {
	Hash       string          `json:"hash"`
	References References      `json:"references"` // used for objects that reference other objects, e.g. schemas
	Metadata   json.RawMessage `json:"metadata"`
}

// we'll keep Reference as a struct for future extensibility at the cost of increased storage space
type Reference struct {
	Name string `json:"name"`
}

type References []Reference
type Directory map[string]ObjectRef

func (r References) Contains(name string) bool {
	for _, ref := range r {
		if ref.Name == name {
			return true
		}
	}
	return false
}

func DirectoryToJSON(directory Directory) ([]byte, error) {
	return json.Marshal(directory)
}

func JSONToDirectory(data []byte) (Directory, error) {
	var directory Directory
	err := json.Unmarshal(data, &directory)
	return directory, err
}

type DirectoryObjectDeleteOptionsSetter interface {
	ReplaceReferencesWithAncestor(bool)
	IgnoreReferences(bool)
	DeleteReferences(bool)
}

type DirectoryObjectDeleteOptions func(DirectoryObjectDeleteOptionsSetter)

func ReplaceReferencesWithAncestor(b bool) DirectoryObjectDeleteOptions {
	return func(s DirectoryObjectDeleteOptionsSetter) {
		s.ReplaceReferencesWithAncestor(b)
	}
}

func IgnoreReferences(b bool) DirectoryObjectDeleteOptions {
	return func(s DirectoryObjectDeleteOptionsSetter) {
		s.IgnoreReferences(b)
	}
}

func DeleteReferences(b bool) DirectoryObjectDeleteOptions {
	return func(s DirectoryObjectDeleteOptionsSetter) {
		s.DeleteReferences(b)
	}
}

/*
Directory is a json that has the following format:
{
	"<path>" : {
		"hash": "<hash>"
	}
	...
}
Here path is the path of the object in the form of /a/b/c/d
*/
