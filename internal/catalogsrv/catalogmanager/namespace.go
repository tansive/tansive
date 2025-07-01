package catalogmanager

import (
	"context"
	"errors"
	"reflect"

	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	schemaerr "github.com/tansive/tansive/internal/catalogsrv/schema/errors"
	"github.com/tansive/tansive/internal/catalogsrv/schema/schemavalidator"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

type NamespaceManager interface {
	Name() string
	Description() string
	Catalog() string
	Variant() string
	GetNamespaceModel() *models.Namespace
	Save(context.Context) apperrors.Error
	ToJson(context.Context) ([]byte, apperrors.Error)
}

type namespaceSchema struct {
	ApiVersion string            `json:"apiVersion" validate:"required,validateVersion"`
	Kind       string            `json:"kind" validate:"required,kindValidator"`
	Metadata   namespaceMetadata `json:"metadata" validate:"required"`
}

type namespaceMetadata struct {
	Catalog     string `json:"catalog" validate:"omitempty,resourceNameValidator"`
	Variant     string `json:"variant" validate:"omitempty,resourceNameValidator"`
	Name        string `json:"name" validate:"required,resourceNameValidator"`
	Description string `json:"description"`
}

type namespaceManager struct {
	namespace models.Namespace
}

// var _ schemamanager.VariantManager = (*variantManager)(nil)

func NewNamespaceManager(ctx context.Context, resourceJSON []byte, catalog string, variant string) (NamespaceManager, apperrors.Error) {
	if len(resourceJSON) == 0 {
		return nil, ErrInvalidSchema
	}

	ns, err := parseAndValidateNamespaceSchema(resourceJSON)
	if err != nil {
		return nil, err
	}

	if err := validateAndSetParameters(ns, catalog, variant); err != nil {
		return nil, err
	}

	catalogID, err := resolveCatalogID(ctx, ns.Metadata.Catalog)
	if err != nil {
		return nil, err
	}

	variantID, err := resolveVariantID(ctx, catalogID, ns.Metadata.Variant)
	if err != nil {
		return nil, err
	}

	namespace := createNamespaceModel(ns, catalogID, variantID)

	return &namespaceManager{
		namespace: namespace,
	}, nil
}

// parseAndValidateNamespaceSchema parses the JSON and validates the namespace schema
func parseAndValidateNamespaceSchema(resourceJSON []byte) (*namespaceSchema, apperrors.Error) {
	ns := &namespaceSchema{}
	if err := json.Unmarshal(resourceJSON, ns); err != nil {
		return nil, ErrInvalidSchema.Err(err)
	}

	validationErrors := ns.Validate()
	if validationErrors != nil {
		return nil, ErrInvalidSchema.Err(validationErrors)
	}

	return ns, nil
}

// validateAndSetParameters validates and sets catalog and variant parameters
func validateAndSetParameters(ns *namespaceSchema, catalog, variant string) apperrors.Error {
	if catalog != "" {
		if err := schemavalidator.V().Var(catalog, "resourceNameValidator"); err != nil {
			return ErrInvalidCatalog
		}
		ns.Metadata.Catalog = catalog
	}

	if variant != "" {
		if err := schemavalidator.V().Var(variant, "resourceNameValidator"); err != nil {
			return ErrInvalidVariant
		}
		ns.Metadata.Variant = variant
	}

	return nil
}

// resolveCatalogID resolves the catalog ID from context or database lookup
func resolveCatalogID(ctx context.Context, catalogName string) (uuid.UUID, apperrors.Error) {
	catalogID := catcommon.GetCatalogID(ctx)

	if catalogID == uuid.Nil || catalogName != catcommon.GetCatalog(ctx) {
		var err apperrors.Error
		catalogID, err = db.DB(ctx).GetCatalogIDByName(ctx, catalogName)
		if err != nil {
			if errors.Is(err, dberror.ErrNotFound) {
				return uuid.Nil, ErrCatalogNotFound
			}
			log.Ctx(ctx).Error().Err(err).Msg("failed to load catalog")
			return uuid.Nil, err
		}
	}

	return catalogID, nil
}

// resolveVariantID resolves the variant ID from context or database lookup
func resolveVariantID(ctx context.Context, catalogID uuid.UUID, variantName string) (uuid.UUID, apperrors.Error) {
	variantID := catcommon.GetVariantID(ctx)

	if variantID == uuid.Nil || variantName != catcommon.GetVariant(ctx) {
		var err apperrors.Error
		variantID, err = db.DB(ctx).GetVariantIDFromName(ctx, catalogID, variantName)
		if err != nil {
			if errors.Is(err, dberror.ErrNotFound) {
				return uuid.Nil, ErrVariantNotFound
			}
			log.Ctx(ctx).Error().Err(err).Msg("failed to load variant")
			return uuid.Nil, err
		}
	}

	return variantID, nil
}

// createNamespaceModel creates a namespace model from the schema and resolved IDs
func createNamespaceModel(ns *namespaceSchema, catalogID, variantID uuid.UUID) models.Namespace {
	return models.Namespace{
		Description: ns.Metadata.Description,
		VariantID:   variantID,
		CatalogID:   catalogID,
		Name:        ns.Metadata.Name,
		Catalog:     ns.Metadata.Catalog,
		Variant:     ns.Metadata.Variant,
		Info:        nil,
	}
}

func (ns *namespaceSchema) Validate() schemaerr.ValidationErrors {
	var validationErrors schemaerr.ValidationErrors
	if ns.Kind != catcommon.NamespaceKind {
		validationErrors = append(validationErrors, schemaerr.ErrUnsupportedKind("kind"))
	}

	err := schemavalidator.V().Struct(ns)
	if err == nil {
		return validationErrors
	}

	validatorErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return append(validationErrors, schemaerr.ErrInvalidSchema)
	}

	value := reflect.ValueOf(ns).Elem()
	typeOfCS := value.Type()

	for _, e := range validatorErrors {
		jsonFieldName := schemavalidator.GetJSONFieldPath(value, typeOfCS, e.StructField())

		switch e.Tag() {
		case "required":
			validationErrors = append(validationErrors, schemaerr.ErrMissingRequiredAttribute(jsonFieldName))
		case "kindValidator":
			validationErrors = append(validationErrors, schemaerr.ErrUnsupportedKind(jsonFieldName))
		case "resourceNameValidator":
			validationErrors = append(validationErrors, schemaerr.ErrInvalidNameFormat(jsonFieldName, e.Value().(string)))
		case "resourcePathValidator":
			validationErrors = append(validationErrors, schemaerr.ErrInvalidObjectPath(jsonFieldName))
		case "validateVersion":
			validationErrors = append(validationErrors, schemaerr.ErrInvalidVersion(jsonFieldName))
		default:
			validationErrors = append(validationErrors, schemaerr.ErrValidationFailed(jsonFieldName))
		}
	}
	return validationErrors
}

func (nm *namespaceManager) Name() string {
	return nm.namespace.Name
}

func (nm *namespaceManager) Description() string {
	return nm.namespace.Description
}

func (nm *namespaceManager) VariantID() uuid.UUID {
	return nm.namespace.VariantID
}

func (nm *namespaceManager) CatalogID() uuid.UUID {
	return nm.namespace.CatalogID
}

func (nm *namespaceManager) Catalog() string {
	return nm.namespace.Catalog
}

func (nm *namespaceManager) Variant() string {
	return nm.namespace.Variant
}

func (nm *namespaceManager) GetNamespaceModel() *models.Namespace {
	return &nm.namespace
}

func LoadNamespaceManagerByName(ctx context.Context, variantID uuid.UUID, name string) (NamespaceManager, apperrors.Error) {
	if variantID == uuid.Nil {
		return nil, ErrInvalidVariant
	}
	namespace, err := db.DB(ctx).GetNamespace(ctx, name, variantID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return nil, ErrNamespaceNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to load namespace")
		return nil, ErrCatalogError.Msg("unable to load namespace")
	}
	return &namespaceManager{
		namespace: *namespace,
	}, nil
}

func (nm *namespaceManager) Save(ctx context.Context) apperrors.Error {
	err := db.DB(ctx).CreateNamespace(ctx, &nm.namespace)
	if err != nil {
		if errors.Is(err, dberror.ErrAlreadyExists) {
			return ErrAlreadyExists.Msg("namespace already exists")
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to create namespace")
		return ErrCatalogError.Msg("unable to create namespace")
	}
	return nil
}

func (nm *namespaceManager) ToJson(ctx context.Context) ([]byte, apperrors.Error) {
	ns := &namespaceSchema{
		ApiVersion: "0.1.0-alpha.1",
		Kind:       catcommon.NamespaceKind,
		Metadata: namespaceMetadata{
			Catalog:     nm.namespace.Catalog,
			Variant:     nm.namespace.Variant,
			Name:        nm.namespace.Name,
			Description: nm.namespace.Description,
		},
	}

	jsonData, e := json.Marshal(ns)
	if e != nil {
		log.Ctx(ctx).Error().Err(e).Msg("unable to marshal workspace schema")
		return nil, ErrCatalogError.Msg("unable to marshal workspace schema")
	}

	return jsonData, nil
}

func DeleteNamespace(ctx context.Context, name string, variantID uuid.UUID) apperrors.Error {
	// check if the namespace exists by retrieving it
	_, err := db.DB(ctx).GetNamespace(ctx, name, variantID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return ErrNamespaceNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to delete namespace")
		return err
	}

	// delete the namespace
	err = db.DB(ctx).DeleteNamespace(ctx, name, variantID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return ErrNamespaceNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to delete namespace")
		return err
	}

	//get the resource directory for the variant
	variant, err := db.DB(ctx).GetVariantByID(ctx, variantID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get variant")
		return err
	}

	dir := variant.ResourceDirectoryID

	// delete the namespace objects in all directories
	_, err = db.DB(ctx).DeleteNamespaceObjects(ctx, catcommon.CatalogObjectTypeResource, dir, name)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to delete namespace objects in Resource")
	}

	return nil
}

type namespaceKind struct {
	req interfaces.RequestContext
	nm  NamespaceManager
}

func (n *namespaceKind) Name() string {
	return n.req.Namespace
}

func (n *namespaceKind) ID() uuid.UUID {
	return n.req.VariantID
}

func (n *namespaceKind) Location() string {
	return "/namespaces/" + n.req.Namespace
}

func (n *namespaceKind) Manager() NamespaceManager {
	return n.nm
}

func (n *namespaceKind) Create(ctx context.Context, resourceJSON []byte) (string, apperrors.Error) {
	nm, err := NewNamespaceManager(ctx, resourceJSON, n.req.Catalog, n.req.Variant)
	if err != nil {
		return "", err
	}
	if err := nm.Save(ctx); err != nil {
		return "", err
	}
	n.req.Namespace = nm.Name()
	if n.req.Catalog == "" {
		n.req.Catalog = nm.Catalog()
	}
	if n.req.Variant == "" {
		n.req.Variant = nm.Variant()
	}
	return n.Location(), nil
}

func (n *namespaceKind) Get(ctx context.Context) ([]byte, apperrors.Error) {
	if n.req.VariantID == uuid.Nil || n.req.Namespace == "" {
		return nil, ErrInvalidNamespace
	}
	namespace, err := LoadNamespaceManagerByName(ctx, n.req.VariantID, n.req.Namespace)
	if err != nil {
		if errors.Is(err, ErrNamespaceNotFound) {
			return nil, ErrNamespaceNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to load namespace")
		return nil, ErrUnableToLoadObject.Msg("unable to load namespace")
	}
	jsonData, err := namespace.ToJson(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to marshal namespace schema")
		return nil, ErrUnableToLoadObject.Msg("unable to marshal namespace schema")
	}
	n.nm = namespace
	return jsonData, nil
}

func (n *namespaceKind) Delete(ctx context.Context) apperrors.Error {
	// delete the namespace
	return DeleteNamespace(ctx, n.req.Namespace, n.req.VariantID)
}

func (n *namespaceKind) Update(ctx context.Context, rsrcJson []byte) apperrors.Error {
	ns := &namespaceSchema{}
	if err := json.Unmarshal(rsrcJson, ns); err != nil {
		return ErrInvalidSchema.Err(err)
	}
	ves := ns.Validate()
	if ves != nil {
		return ErrInvalidSchema.Err(ves)
	}
	_, err := n.Get(ctx)
	if err != nil {
		return err
	}
	namespace := n.nm.GetNamespaceModel()
	if namespace == nil {
		return ErrInvalidNamespace
	}
	namespace.Description = ns.Metadata.Description
	namespace.Name = ns.Metadata.Name
	err = db.DB(ctx).UpdateNamespace(ctx, namespace)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return ErrNamespaceNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to update namespace")
		return ErrUnableToLoadObject.Msg("unable to update namespace")
	}
	n.req.Namespace = namespace.Name
	return nil
}

func (n *namespaceKind) List(ctx context.Context) ([]byte, apperrors.Error) {
	return nil, nil
}

func NewNamespaceKindHandler(ctx context.Context, reqCtx interfaces.RequestContext) (interfaces.KindHandler, apperrors.Error) {
	if reqCtx.Catalog == "" || reqCtx.CatalogID == uuid.Nil {
		return nil, ErrInvalidCatalog
	}
	if reqCtx.Variant == "" || reqCtx.VariantID == uuid.Nil {
		return nil, ErrInvalidVariant
	}
	return &namespaceKind{
		req: reqCtx,
	}, nil
}
