package catalogmanager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"reflect"
	"slices"

	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/objectstore"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	schemaerr "github.com/tansive/tansive/internal/catalogsrv/schema/errors"
	"github.com/tansive/tansive/internal/catalogsrv/schema/schemavalidator"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/pkg/types"
	"github.com/tidwall/gjson"
)

// Resource represents a single resource in the catalog system.
// It contains metadata, schema, and value information.
type Resource struct {
	ApiVersion string              `json:"apiVersion" validate:"required,validateVersion"`
	Kind       string              `json:"kind" validate:"required,oneof=Resource"`
	Metadata   interfaces.Metadata `json:"metadata" validate:"required"`
	Spec       ResourceSpec        `json:"spec,omitempty"` // we can have empty collections
}

// ResourceSpec defines the specification for a resource, including its schema,
// value, policy, and annotations.
type ResourceSpec struct {
	Provider    ResourceProvider       `json:"-" validate:"required_without=Schema,omitempty,resourceNameValidator"`
	Schema      json.RawMessage        `json:"schema" validate:"required_without=Provider,omitempty"`
	Value       types.NullableAny      `json:"value" validate:"omitempty"`
	Annotations interfaces.Annotations `json:"annotations" validate:"omitempty,dive,keys,noSpaces,endkeys"`
}

// ResourceProvider is a placeholder for the resource provider.
type ResourceProvider struct {
	_ any `json:"-"`
}

// JSON returns the JSON representation of the resource.
func (r *Resource) JSON(ctx context.Context) ([]byte, apperrors.Error) {
	j, err := json.Marshal(r)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to marshal object schema")
		return j, ErrUnableToLoadObject
	}
	return j, nil
}

// Validate performs validation on the resource, including:
// - Kind validation
// - Schema validation
// - Value validation against the schema
func (r *Resource) Validate() schemaerr.ValidationErrors {
	var validationErrors schemaerr.ValidationErrors
	if r.Kind != catcommon.ResourceKind {
		validationErrors = append(validationErrors, schemaerr.ErrUnsupportedKind("kind"))
	}

	err := schemavalidator.V().Struct(r)
	if err == nil {
		if len(r.Spec.Schema) > 0 {
			compiledSchema, err := compileSchema(string(r.Spec.Schema))
			if err != nil {
				validationErrors = append(validationErrors, schemaerr.ErrValidationFailed(fmt.Sprintf("resource %s: %v", r.Metadata.Name, err)))
			}
			if compiledSchema != nil {
				// validate the value against the schema
				if err := r.ValidateValue(r.Spec.Value, compiledSchema); err != nil {
					validationErrors = append(validationErrors, schemaerr.ErrValidationFailed(fmt.Sprintf("resource %s: %v", r.Metadata.Name, err)))
				}
			}
		}
		return validationErrors
	}

	validatorErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return append(validationErrors, schemaerr.ErrInvalidSchema)
	}

	value := reflect.ValueOf(r).Elem()
	typeOfCS := value.Type()

	for _, e := range validatorErrors {
		jsonFieldName := schemavalidator.GetJSONFieldPath(value, typeOfCS, e.StructField())
		switch e.Tag() {
		case "required":
			validationErrors = append(validationErrors, schemaerr.ErrMissingRequiredAttribute(jsonFieldName))
		case "oneof":
			validationErrors = append(validationErrors, schemaerr.ErrInvalidFieldSchema(jsonFieldName, e.Value().(string)))
		case "resourceNameValidator":
			val, _ := e.Value().(string)
			validationErrors = append(validationErrors, schemaerr.ErrInvalidNameFormat(jsonFieldName, val))
		case "resourcePathValidator":
			validationErrors = append(validationErrors, schemaerr.ErrInvalidObjectPath(jsonFieldName))
		default:
			val := e.Value()
			param := e.Param()
			s := fmt.Sprintf("%v: %v", param, val)
			validationErrors = append(validationErrors, schemaerr.ErrValidationFailed(s))
		}
	}
	return validationErrors
}

// ValidateValue validates a value against the resource's JSON schema.
// It accepts an optional pre-compiled schema to avoid recompilation.
func (r *Resource) ValidateValue(value types.NullableAny, optsCompiledSchema ...*jsonschema.Schema) error {
	var compiledSchema *jsonschema.Schema
	var err error
	if len(optsCompiledSchema) == 0 {
		compiledSchema, err = compileSchema(string(r.Spec.Schema))
		if err != nil {
			return fmt.Errorf("failed to compile schema: %w", err)
		}
	} else {
		compiledSchema = optsCompiledSchema[0]
	}
	if compiledSchema == nil {
		return fmt.Errorf("failed to compile schema")
	}

	// Handle nil values - only reject if schema doesn't allow null
	if value.IsNil() {
		// Check if schema allows null type
		if !slices.Contains(compiledSchema.Types, "null") {
			return fmt.Errorf("value cannot be nil")
		}
		return nil
	}

	return compiledSchema.Validate(value.Get())
}

// compileSchema compiles a JSON schema string into a jsonschema.Schema.
// It validates the schema is valid JSON and handles self-referential schemas.
func compileSchema(schema string) (*jsonschema.Schema, error) {
	// First validate that the schema is valid JSON using gjson
	if !gjson.Valid(schema) {
		return nil, fmt.Errorf("invalid JSON schema")
	}

	compiler := jsonschema.NewCompiler()
	// Allow schemas with $id to refer to themselves
	compiler.LoadURL = func(url string) (io.ReadCloser, error) {
		if url == "inline://schema" {
			return io.NopCloser(bytes.NewReader([]byte(schema))), nil
		}
		return nil, fmt.Errorf("unsupported schema ref: %s", url)
	}
	err := compiler.AddResource("inline://schema", bytes.NewReader([]byte(schema)))
	if err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}
	compiledSchema, err := compiler.Compile("inline://schema")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return compiledSchema, nil
}

// resourceManager implements the ResourceManager interface for managing a single resource.
type resourceManager struct {
	resource Resource
}

// Metadata returns the resource's metadata.
func (rm *resourceManager) Metadata() interfaces.Metadata {
	return rm.resource.Metadata
}

// FullyQualifiedName returns the fully qualified name of the resource.
func (rm *resourceManager) FullyQualifiedName() string {
	m := rm.resource.Metadata
	return path.Clean(m.Path + "/" + m.Name)
}

// SetValue sets the resource's value after validating it against the schema.
func (rm *resourceManager) SetValue(ctx context.Context, value types.NullableAny) apperrors.Error {
	// validate the value against the schema
	if err := rm.resource.ValidateValue(value); err != nil {
		return ErrInvalidResourceValue.Msg(err.Error())
	}
	rm.resource.Spec.Value = value
	return nil
}

// GetValue returns the resource's current value.
func (rm *resourceManager) GetValue(ctx context.Context) types.NullableAny {
	return rm.resource.Spec.Value
}

// GetValueJSON returns the JSON representation of the resource's value.
func (rm *resourceManager) GetValueJSON(ctx context.Context) ([]byte, apperrors.Error) {
	json, err := json.Marshal(rm.resource.Spec.Value)
	if err != nil {
		return nil, ErrInvalidResourceValue.Msg("unable to obtain resource value")
	}
	return json, nil
}

// StorageRepresentation returns the object storage representation of the resource.
func (rm *resourceManager) StorageRepresentation() *objectstore.ObjectStorageRepresentation {
	s := objectstore.ObjectStorageRepresentation{
		Version: rm.resource.ApiVersion,
		Type:    catcommon.CatalogObjectTypeResource,
	}
	s.Spec, _ = json.Marshal(rm.resource.Spec)
	s.Description = rm.resource.Metadata.Description
	s.Entropy = rm.resource.Metadata.GetEntropyBytes(catcommon.CatalogObjectTypeResource)
	return &s
}

// GetStoragePath returns the storage path for the resource.
func (rm *resourceManager) GetStoragePath() string {
	m := rm.Metadata()
	return getResourceStoragePath(&m)
}

// getResourceStoragePath constructs the storage path for a resource based on its metadata.
func getResourceStoragePath(m *interfaces.Metadata) string {
	t := catcommon.CatalogObjectTypeResource
	rsrcPath := m.GetStoragePath(t)
	pathWithName := path.Clean(rsrcPath + "/" + m.Name)
	return pathWithName
}

// Save saves the resource to the database.
// It handles the creation or update of both the resource and its associated catalog object.
func (rm *resourceManager) Save(ctx context.Context) apperrors.Error {
	if rm == nil {
		return ErrEmptySchema
	}

	t := catcommon.CatalogObjectTypeResource

	m := rm.Metadata()
	s := rm.StorageRepresentation()
	storagePath := rm.GetStoragePath()

	data, err := s.Serialize()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to serialize resource")
		return err
	}
	newHash := s.GetHash()

	// Store this object and update the reference
	obj := models.CatalogObject{
		Type:    t,
		Hash:    newHash,
		Data:    data,
		Version: rm.resource.ApiVersion,
	}

	// Get the directory ID for the resource
	catalogID := catcommon.GetCatalogID(ctx)
	if catalogID == uuid.Nil {
		var err apperrors.Error
		catalogID, err = db.DB(ctx).GetCatalogIDByName(ctx, m.Catalog)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("catalog", m.Catalog).Msg("Failed to get catalog ID by name")
			return err
		}
	}

	variant, err := db.DB(ctx).GetVariant(ctx, catalogID, uuid.Nil, m.Variant.String())
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("catalogID", catalogID.String()).Str("name", m.Name).Msg("Failed to get variant")
		return err
	}

	// Create the resource model
	rsrc := &models.Resource{
		Path:      storagePath,
		Hash:      newHash,
		VariantID: variant.VariantID,
	}

	// Store the object
	err = db.DB(ctx).UpsertResourceObject(ctx, rsrc, &obj, variant.ResourceDirectoryID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("path", storagePath).Msg("Failed to store object")
		return err
	}

	return nil
}

// DeleteResource deletes a resource from the database.
func DeleteResource(ctx context.Context, m *interfaces.Metadata) apperrors.Error {
	if m == nil {
		return ErrInvalidObject.Msg("unable to infer object metadata")
	}

	// Get the directory ID for the resource
	catalogID := catcommon.GetCatalogID(ctx)
	if catalogID == uuid.Nil {
		var err apperrors.Error
		catalogID, err = db.DB(ctx).GetCatalogIDByName(ctx, m.Catalog)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("catalog", m.Catalog).Msg("Failed to get catalog ID by name")
			return err
		}
	}

	variant, err := db.DB(ctx).GetVariant(ctx, catalogID, uuid.Nil, m.Variant.String())
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("catalogID", catalogID.String()).Str("name", m.Name).Msg("Failed to get variant")
		return err
	}

	pathWithName := path.Clean(m.GetStoragePath(catcommon.CatalogObjectTypeResource) + "/" + m.Name)

	// Delete the resource
	hash, err := db.DB(ctx).DeleteResource(ctx, pathWithName, variant.ResourceDirectoryID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return ErrObjectNotFound
		}
		log.Ctx(ctx).Error().Err(err).Str("path", pathWithName).Msg("Failed to delete object")
		return err
	}

	if hash != "" {
		err = db.DB(ctx).DeleteCatalogObject(ctx, catcommon.CatalogObjectTypeResource, hash)
		if !errors.Is(err, dberror.ErrNotFound) {
			log.Ctx(ctx).Error().Err(err).Str("hash", string(hash)).Msg("failed to delete object from database")
		}
	} else {
		log.Ctx(ctx).Warn().Str("path", pathWithName).Msg("resource not found")
		return ErrObjectNotFound
	}

	return nil
}

// JSON returns the JSON representation of the resource.
func (rm *resourceManager) JSON(ctx context.Context) ([]byte, apperrors.Error) {
	return rm.resource.JSON(ctx)
}

func (rm *resourceManager) SpecJSON(ctx context.Context) ([]byte, apperrors.Error) {
	j, err := json.Marshal(rm.resource.Spec)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to marshal resource spec")
		return nil, ErrInvalidResourceDefinition
	}
	return j, nil
}
