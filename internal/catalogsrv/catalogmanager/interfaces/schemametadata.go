package interfaces

import (
	stdjson "encoding/json"
	"path"
	"reflect"
	"strings"

	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	schemaerr "github.com/tansive/tansive/internal/catalogsrv/schema/errors"
	"github.com/tansive/tansive/internal/catalogsrv/schema/schemavalidator"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/pkg/types"
)

// Modifying this struct should also change the json
type Metadata struct {
	Name        string               `json:"name" validate:"required,resourceNameValidator"`
	Catalog     string               `json:"catalog" validate:"required,resourceNameValidator"`
	Variant     types.NullableString `json:"variant,omitempty" validate:"resourceNameValidator"`
	Namespace   types.NullableString `json:"namespace,omitempty" validate:"omitempty,resourceNameValidator"`
	Path        string               `json:"path,omitempty" validate:"omitempty,resourcePathValidator"`
	Description string               `json:"description"`
	IDS         IDS                  `json:"-"`
}

type IDS struct {
	CatalogID uuid.UUID
	VariantID uuid.UUID
}

var _ stdjson.Marshaler = Metadata{}
var _ stdjson.Marshaler = &Metadata{}

func (rs *Metadata) Validate() schemaerr.ValidationErrors {
	var ves schemaerr.ValidationErrors
	err := schemavalidator.V().Struct(rs)
	if err == nil {
		return nil
	}
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return append(ves, schemaerr.ErrInvalidSchema)
	}

	value := reflect.ValueOf(rs).Elem()
	typeOfCS := value.Type()

	for _, e := range ve {
		jsonFieldName := schemavalidator.GetJSONFieldPath(value, typeOfCS, e.StructField())
		jsonFieldName = "metadata." + jsonFieldName
		switch e.Tag() {
		case "required":
			ves = append(ves, schemaerr.ErrMissingRequiredAttribute(jsonFieldName))
		case "resourceNameValidator":
			val, _ := e.Value().(string)
			ves = append(ves, schemaerr.ErrInvalidNameFormat(jsonFieldName, val))
		case "resourcePathValidator":
			ves = append(ves, schemaerr.ErrInvalidObjectPath(jsonFieldName))
		default:
			ves = append(ves, schemaerr.ErrValidationFailed(jsonFieldName))
		}
	}
	return ves
}

func (s Metadata) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)

	m["name"] = s.Name
	m["catalog"] = s.Catalog
	m["description"] = s.Description

	if s.Variant.Valid {
		m["variant"] = s.Variant.Value
	}
	if s.Namespace.Valid {
		m["namespace"] = s.Namespace.Value
	}
	if s.Path != "" {
		m["path"] = s.Path
	}

	return json.Marshal(m)
}

func (m Metadata) GetStoragePath(t catcommon.CatalogObjectType) string {
	_ = t // unused
	if m.Namespace.IsNil() {
		return path.Clean("/" + catcommon.DefaultNamespace + "/" + m.Path)
	} else {
		return path.Clean("/" + catcommon.DefaultNamespace + "/" + m.Namespace.String() + "/" + m.Path)
	}
}

func (m Metadata) GetEntropyBytes(t catcommon.CatalogObjectType) []byte {
	entropy := m.Catalog + ":" + string(t)
	return []byte(entropy)
}

func (m Metadata) GetFullyQualifiedName() string {
	return path.Clean(m.Path + "/" + m.Name)
}

func (m *Metadata) SetNameAndPathFromStoragePath(storagePath string) {
	m.Name = path.Base(storagePath)
	m.Path = path.Dir(storagePath)
	m.Path = strings.TrimPrefix(m.Path, "/"+catcommon.DefaultNamespace)
	m.Path = "/" + strings.TrimPrefix(m.Path, "/")
}
