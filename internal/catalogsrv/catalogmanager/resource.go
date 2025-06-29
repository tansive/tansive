package catalogmanager

import (
	"context"
	"net/url"
	"path"

	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/objectstore"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/pkg/types"
)

type ResourceManager interface {
	Metadata() interfaces.Metadata
	FullyQualifiedName() string
	GetValue(ctx context.Context) types.NullableAny
	GetValueJSON(ctx context.Context) ([]byte, apperrors.Error)
	SetValue(ctx context.Context, value types.NullableAny) apperrors.Error
	StorageRepresentation() *objectstore.ObjectStorageRepresentation
	Save(ctx context.Context) apperrors.Error
	GetStoragePath() string
	JSON(ctx context.Context) ([]byte, apperrors.Error)
	SpecJSON(ctx context.Context) ([]byte, apperrors.Error)
}

// NewResourceManager creates a new ResourceManager instance from the provided JSON schema and metadata.
// It validates the schema and metadata before creating the manager.
func NewResourceManager(ctx context.Context, rsrcJSON []byte, m *interfaces.Metadata) (ResourceManager, apperrors.Error) {
	if len(rsrcJSON) == 0 {
		return nil, ErrEmptySchema
	}

	// Get the metadata, replace fields in JSON from provided metadata, and set defaults.
	rsrcJSON, m, err := canonicalizeMetadata(rsrcJSON, catcommon.ResourceKind, m)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to canonicalize metadata")
		return nil, ErrSchemaSerialization
	}

	var rsrc Resource
	if err := json.Unmarshal(rsrcJSON, &rsrc); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal resource")
		return nil, ErrSchemaValidation
	}

	if validationErrs := rsrc.Validate(); validationErrs != nil {
		log.Ctx(ctx).Error().Err(validationErrs).Msg("Resource validation failed")
		return nil, ErrSchemaValidation.Msg(validationErrs.Error())
	}

	rsrc.Metadata = *m

	return &resourceManager{resource: rsrc}, nil
}

func LoadResourceManagerByHash(ctx context.Context, hash string, m *interfaces.Metadata) (ResourceManager, apperrors.Error) {
	// get the object from catalog object store
	obj, err := db.DB(ctx).GetCatalogObject(ctx, hash)
	if err != nil {
		return nil, err
	}
	return resourceManagerFromObject(ctx, obj, m)
}

// LoadResourceManagerByPath loads a resource manager from the database by path.
func LoadResourceManagerByPath(ctx context.Context, m *interfaces.Metadata) (ResourceManager, apperrors.Error) {
	if m == nil {
		return nil, ErrInvalidObject.Msg("unable to infer object metadata")
	}

	// Get the directory ID for the resource
	catalogID := catcommon.GetCatalogID(ctx)
	var err apperrors.Error

	if catalogID == uuid.Nil {
		catalogID, err = db.DB(ctx).GetCatalogIDByName(ctx, m.Catalog)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("catalog", m.Catalog).Msg("Failed to get catalog ID by name")
			return nil, err
		}
	}

	variant, err := db.DB(ctx).GetVariant(ctx, catalogID, uuid.Nil, m.Variant.String())
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("catalogID", catalogID.String()).Str("name", m.Name).Msg("Failed to get variant")
		return nil, err
	}

	pathWithName := path.Clean(m.GetStoragePath(catcommon.CatalogObjectTypeResource) + "/" + m.Name)

	obj, err := db.DB(ctx).GetResourceObject(ctx, pathWithName, variant.ResourceDirectoryID)
	if err != nil {
		return nil, err
	}

	return resourceManagerFromObject(ctx, obj, m)
}

func resourceManagerFromObject(ctx context.Context, obj *models.CatalogObject, m *interfaces.Metadata) (ResourceManager, apperrors.Error) {
	if obj == nil {
		return nil, ErrEmptySchema
	}

	var storageRep objectstore.ObjectStorageRepresentation
	if err := json.Unmarshal(obj.Data, &storageRep); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal resource")
		return nil, ErrSchemaValidation
	}

	if storageRep.Type != catcommon.CatalogObjectTypeResource {
		log.Ctx(ctx).Error().Msg("Invalid type")
		return nil, ErrUnableToLoadObject
	}

	rm := &resourceManager{}
	if err := json.Unmarshal(storageRep.Spec, &rm.resource.Spec); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal resource schema spec")
		return nil, ErrUnableToLoadObject
	}

	rm.resource.Kind = catcommon.ResourceKind
	rm.resource.ApiVersion = storageRep.Version
	rm.resource.Metadata = *m
	rm.resource.Metadata.Description = storageRep.Description

	return rm, nil
}

var _ interfaces.KindHandler = &resourceKindHandler{}

// resourceKindHandler implements the KindHandler interface for managing individual resources.
// It handles CRUD operations for resources and maintains the request context.
type resourceKindHandler struct {
	req interfaces.RequestContext
	rm  ResourceManager
}

// Name returns the name of the resource from the request context.
func (h *resourceKindHandler) Name() string {
	return h.req.ObjectName
}

// Location returns the fully qualified path to the resource, including any query parameters.
// The path is constructed using the resource name and namespace (if present).
func (h *resourceKindHandler) Location() string {
	objName := catcommon.KindNameFromObjectType(h.req.ObjectType)
	loc := path.Clean("/" + objName + h.rm.FullyQualifiedName())

	q := url.Values{}
	if namespace := h.rm.Metadata().Namespace.String(); namespace != "" {
		q.Set("namespace", namespace)
	}

	if qStr := q.Encode(); qStr != "" {
		loc += "?" + qStr
	}

	return loc
}

// Manager returns the underlying ResourceManager instance.
func (h *resourceKindHandler) Manager() ResourceManager {
	return h.rm
}

// Create creates a new resource from the provided JSON data.
// It validates the input, saves the resource, and updates the request context with the new resource's metadata.
func (h *resourceKindHandler) Create(ctx context.Context, rsrcJSON []byte) (string, apperrors.Error) {
	m := &interfaces.Metadata{
		Catalog:   h.req.Catalog,
		Variant:   types.NullableStringFrom(h.req.Variant),
		Namespace: types.NullableStringFrom(h.req.Namespace),
	}

	rm, err := NewResourceManager(ctx, rsrcJSON, m)
	if err != nil {
		return "", err
	}

	if err := rm.Save(ctx); err != nil {
		return "", err
	}

	h.req.ObjectName = rm.Metadata().Name
	h.req.ObjectPath = rm.Metadata().Path
	h.req.ObjectType = catcommon.CatalogObjectTypeResource
	h.rm = rm

	// Update request context with metadata if not set
	if h.req.Catalog == "" {
		h.req.Catalog = rm.Metadata().Catalog
	}
	if h.req.Variant == "" {
		h.req.Variant = rm.Metadata().Variant.String()
	}
	if h.req.Namespace == "" {
		h.req.Namespace = rm.Metadata().Namespace.String()
	}

	return h.Location(), nil
}

// Get retrieves a resource by its path and returns it as JSON.
// It validates the metadata and loads the resource from storage.
func (h *resourceKindHandler) Get(ctx context.Context) ([]byte, apperrors.Error) {
	m := &interfaces.Metadata{
		Catalog:   h.req.Catalog,
		Variant:   types.NullableStringFrom(h.req.Variant),
		Namespace: types.NullableStringFrom(h.req.Namespace),
		Path:      h.req.ObjectPath,
		Name:      h.req.ObjectName,
	}

	if err := m.Validate(); err != nil {
		return nil, ErrSchemaValidation.Msg(err.Error())
	}

	rm, err := LoadResourceManagerByPath(ctx, m)
	if err != nil {
		return nil, err
	}
	switch h.req.ObjectProperty {
	case catcommon.ResourcePropertyDefinition:
		return rm.JSON(ctx)
	case catcommon.ResourcePropertyValue:
		return rm.GetValueJSON(ctx)
	default:
		return nil, ErrDisallowedByPolicy
	}
}

// Update updates an existing resource with new data.
// It validates the input, checks for the resource's existence, and saves the changes.
func (h *resourceKindHandler) Update(ctx context.Context, rsrcJSON []byte) apperrors.Error {
	m := &interfaces.Metadata{
		Catalog:   h.req.Catalog,
		Variant:   types.NullableStringFrom(h.req.Variant),
		Path:      h.req.ObjectPath,
		Name:      h.req.ObjectName,
		Namespace: types.NullableStringFrom(h.req.Namespace),
	}

	if err := m.Validate(); err != nil {
		return ErrSchemaValidation.Msg(err.Error())
	}

	// Load the existing object
	existing, err := LoadResourceManagerByPath(ctx, m)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrObjectNotFound
	}

	switch h.req.ObjectProperty {
	case catcommon.ResourcePropertyDefinition:
		rm, err := NewResourceManager(ctx, rsrcJSON, m)
		if err != nil {
			return err
		}
		return rm.Save(ctx)
	case catcommon.ResourcePropertyValue:
		val := types.NullableAny{}
		if err := json.Unmarshal(rsrcJSON, &val); err != nil {
			return ErrInvalidResourceValue
		}
		if err := existing.SetValue(ctx, val); err != nil {
			return err
		}
		return existing.Save(ctx)
	default:
		return ErrDisallowedByPolicy
	}
}

// Delete removes a resource from storage.
// It validates the metadata and deletes the resource if it exists.
func (h *resourceKindHandler) Delete(ctx context.Context) apperrors.Error {
	m := &interfaces.Metadata{
		Catalog:   h.req.Catalog,
		Variant:   types.NullableStringFrom(h.req.Variant),
		Path:      h.req.ObjectPath,
		Name:      h.req.ObjectName,
		Namespace: types.NullableStringFrom(h.req.Namespace),
	}

	if err := DeleteResource(ctx, m); err != nil {
		pathWithName := path.Clean(m.GetStoragePath(h.req.ObjectType) + "/" + h.req.ObjectName)
		log.Ctx(ctx).Error().Err(err).Str("path", pathWithName).Msg("Failed to delete object")
		return err
	}
	return nil
}

func (h *resourceKindHandler) List(ctx context.Context) ([]byte, apperrors.Error) {
	variant, err := db.DB(ctx).GetVariantByID(ctx, h.req.VariantID)
	if err != nil {
		return nil, ErrInvalidVariant
	}

	resources, err := db.DB(ctx).ListResources(ctx, variant.ResourceDirectoryID)
	if err != nil {
		return nil, ErrCatalogError.Msg("unable to list resources")
	}

	resourceList := make(map[string]json.RawMessage)
	for _, resource := range resources {
		m := &interfaces.Metadata{
			Catalog:   h.req.Catalog,
			Variant:   types.NullableStringFrom(h.req.Variant),
			Namespace: types.NullableStringFrom(h.req.Namespace),
		}
		m.SetNameAndPathFromStoragePath(resource.Path)
		rm, err := LoadResourceManagerByHash(ctx, resource.Hash, m)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("path", resource.Path).Msg("Failed to load resource")
			continue
		}

		j, err := rm.JSON(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("path", resource.Path).Msg("Failed to marshal resource")
			continue
		}
		resourceList[path.Clean(m.Path+"/"+m.Name)] = j
	}

	j, goErr := json.Marshal(resourceList)
	if goErr != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to marshal resource list")
		return nil, ErrInvalidResourceDefinition
	}

	return j, nil
}

func NewResourceKindHandler(ctx context.Context, req interfaces.RequestContext) (interfaces.KindHandler, apperrors.Error) {
	if req.Catalog == "" {
		return nil, ErrInvalidCatalog
	}
	if req.Variant == "" {
		return nil, ErrInvalidVariant
	}
	return &resourceKindHandler{
		req: req,
	}, nil
}
