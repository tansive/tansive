package catalogmanager

import (
	"context"
	"errors"
	"reflect"

	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgtype"
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
	"github.com/tidwall/gjson"
)

type VariantManager interface {
	ID() uuid.UUID
	Name() string
	Description() string
	CatalogID() uuid.UUID
	Save(context.Context) apperrors.Error
	ToJson(context.Context) ([]byte, apperrors.Error)
}

type variantSchema struct {
	ApiVersion string          `json:"apiVersion" validate:"required,validateVersion"`
	Kind       string          `json:"kind" validate:"required,kindValidator"`
	Metadata   variantMetadata `json:"metadata" validate:"required"`
}

type variantMetadata struct {
	Name        string `json:"name" validate:"required,resourceNameValidator"`
	Catalog     string `json:"catalog" validate:"required,resourceNameValidator"`
	Description string `json:"description"`
}

type variantManager struct {
	variant models.Variant
}

var _ VariantManager = (*variantManager)(nil)

func (vs *variantSchema) Validate() schemaerr.ValidationErrors {
	var validationErrors schemaerr.ValidationErrors
	if vs.Kind != catcommon.VariantKind {
		validationErrors = append(validationErrors, schemaerr.ErrUnsupportedKind("kind"))
	}

	err := schemavalidator.V().Struct(vs)
	if err == nil {
		return validationErrors
	}

	validatorErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return append(validationErrors, schemaerr.ErrInvalidSchema)
	}

	value := reflect.ValueOf(vs).Elem()
	typeOfCS := value.Type()

	for _, e := range validatorErrors {
		jsonFieldName := schemavalidator.GetJSONFieldPath(value, typeOfCS, e.StructField())

		switch e.Tag() {
		case "required":
			validationErrors = append(validationErrors, schemaerr.ErrMissingRequiredAttribute(jsonFieldName))
		case "resourceNameValidator":
			val, _ := e.Value().(string)
			validationErrors = append(validationErrors, schemaerr.ErrInvalidNameFormat(jsonFieldName, val))
		case "kindValidator":
			validationErrors = append(validationErrors, schemaerr.ErrUnsupportedKind(jsonFieldName))
		case "validateVersion":
			validationErrors = append(validationErrors, schemaerr.ErrInvalidVersion(jsonFieldName))
		default:
			validationErrors = append(validationErrors, schemaerr.ErrValidationFailed(jsonFieldName))
		}
	}

	return validationErrors
}

func NewVariantManager(ctx context.Context, resourceJSON []byte, name string, catalog string) (VariantManager, apperrors.Error) {
	projectID := catcommon.GetProjectID(ctx)
	if projectID == "" {
		return nil, ErrInvalidProject
	}

	if len(resourceJSON) == 0 {
		return nil, ErrInvalidSchema
	}

	vs := &variantSchema{}
	if err := json.Unmarshal(resourceJSON, vs); err != nil {
		return nil, ErrInvalidSchema.Err(err)
	}
	if !catcommon.IsApiVersionCompatible(vs.ApiVersion) {
		return nil, ErrInvalidVersion
	}
	if vs.Kind != "Variant" {
		return nil, ErrInvalidKind
	}

	// replace name and catalog if not empty
	if name != "" {
		if !schemavalidator.ValidateKindName(name) {
			return nil, ErrInvalidNameFormat
		}
		vs.Metadata.Name = name
	}

	if catalog != "" {
		if !schemavalidator.ValidateKindName(catalog) {
			return nil, ErrInvalidCatalog
		}
		vs.Metadata.Catalog = catalog
	}

	// validate the schema
	validationErrors := vs.Validate()
	if validationErrors != nil {
		return nil, ErrInvalidSchema.Err(validationErrors)
	}

	// Get catalog ID from context or resolve by name
	catalogID := catcommon.GetCatalogID(ctx)
	if catalogID == uuid.Nil {
		var err apperrors.Error
		catalogID, err = db.DB(ctx).GetCatalogIDByName(ctx, vs.Metadata.Catalog)
		if err != nil {
			if errors.Is(err, dberror.ErrNotFound) {
				return nil, ErrCatalogNotFound.New("catalog not found: " + vs.Metadata.Catalog)
			}
			log.Ctx(ctx).Error().Err(err).Msg("failed to load catalog")
			return nil, err
		}
	}

	variant := models.Variant{
		Name:        vs.Metadata.Name,
		Description: vs.Metadata.Description,
		CatalogID:   catalogID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	}

	return &variantManager{
		variant: variant,
	}, nil
}

func (vm *variantManager) ID() uuid.UUID {
	return vm.variant.VariantID
}

func (vm *variantManager) Name() string {
	return vm.variant.Name
}

func (vm *variantManager) Description() string {
	return vm.variant.Description
}

func (vm *variantManager) CatalogID() uuid.UUID {
	return vm.variant.CatalogID
}

func LoadVariantManager(ctx context.Context, catalogID uuid.UUID, variantID uuid.UUID, name string) (VariantManager, apperrors.Error) {
	if variantID == uuid.Nil && (catalogID == uuid.Nil || name == "") {
		return nil, ErrInvalidVariant.Msg("variant ID or both catalog ID and name must be provided")
	}
	variant, err := db.DB(ctx).GetVariant(ctx, catalogID, variantID, name)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return nil, ErrVariantNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to load variant")
		return nil, err
	}
	return &variantManager{
		variant: *variant,
	}, nil
}

func (vm *variantManager) Save(ctx context.Context) apperrors.Error {
	err := db.DB(ctx).CreateVariant(ctx, &vm.variant)
	if err != nil {
		if errors.Is(err, dberror.ErrAlreadyExists) {
			return ErrAlreadyExists.Msg("variant already exists")
		}
		if errors.Is(err, dberror.ErrInvalidCatalog) {
			return ErrInvalidVariant.Msg("catalog does not exist or is invalid")
		}
		if errors.Is(err, dberror.ErrInvalidInput) {
			return ErrInvalidVariant.Msg("invalid variant name format")
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to create variant")
		return err
	}
	return nil
}

func (vm *variantManager) ToJson(ctx context.Context) ([]byte, apperrors.Error) {
	// Get name of the catalog
	catalog, err := db.DB(ctx).GetCatalogByID(ctx, vm.variant.CatalogID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return nil, ErrCatalogNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to load catalog")
		return nil, err
	}

	s := variantSchema{
		ApiVersion: catcommon.ApiVersion,
		Kind:       catcommon.VariantKind,
		Metadata: variantMetadata{
			Name:        vm.variant.Name,
			Catalog:     catalog.Name,
			Description: vm.variant.Description,
		},
	}

	j, e := json.Marshal(s)
	if e != nil {
		log.Ctx(ctx).Error().Err(e).Msg("failed to marshal json")
		return nil, ErrUnableToLoadObject
	}

	return j, nil
}

func DeleteVariant(ctx context.Context, catalogID, variantID uuid.UUID, name string) apperrors.Error {
	err := db.DB(ctx).DeleteVariant(ctx, catalogID, variantID, name)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return ErrVariantNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to delete catalog")
		return err
	}
	return nil
}

// TODO Handle base variant and copy of data

type variantKind struct {
	req interfaces.RequestContext
	vm  VariantManager
}

func (v *variantKind) Name() string {
	return v.req.Variant
}

func (v *variantKind) Location() string {
	return "/variants/" + v.vm.Name()
}

func (v *variantKind) Manager() VariantManager {
	return v.vm
}

func (v *variantKind) Create(ctx context.Context, resourceJSON []byte) (string, apperrors.Error) {
	vm, err := NewVariantManager(ctx, resourceJSON, "", v.req.Catalog)
	if err != nil {
		return "", err
	}

	err = vm.Save(ctx)
	if err != nil {
		return "", err
	}
	v.req.Variant = vm.Name()
	v.req.VariantID = vm.ID()
	v.req.CatalogID = vm.CatalogID()
	v.vm = vm
	v.req.Catalog = gjson.GetBytes(resourceJSON, "metadata.catalog").String()
	return v.Location(), nil
}

func (v *variantKind) Get(ctx context.Context) ([]byte, apperrors.Error) {
	variant, err := LoadVariantManager(ctx, v.req.CatalogID, v.req.VariantID, v.req.Variant)
	if err != nil {
		return nil, err
	}
	return variant.ToJson(ctx)
}

func (v *variantKind) Delete(ctx context.Context) apperrors.Error {
	return DeleteVariant(ctx, v.req.CatalogID, v.req.VariantID, v.req.Variant)
}

func (v *variantKind) Update(ctx context.Context, rsrcJson []byte) apperrors.Error {
	schema := &variantSchema{}
	if err := json.Unmarshal(rsrcJson, schema); err != nil {
		return ErrInvalidSchema.Err(err)
	}

	validationErrors := schema.Validate()
	if validationErrors != nil {
		return ErrInvalidSchema.Err(validationErrors)
	}

	variant, err := db.DB(ctx).GetVariant(ctx, v.req.CatalogID, v.req.VariantID, schema.Metadata.Name)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return ErrVariantNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to load variant")
		return err
	}

	variant.Description = schema.Metadata.Description

	err = db.DB(ctx).UpdateVariant(ctx, uuid.Nil, v.req.Variant, variant)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update variant")
		return ErrUnableToUpdateObject.Msg("failed to update variant")
	}
	return nil
}

func (v *variantKind) List(ctx context.Context) ([]byte, apperrors.Error) {
	return nil, nil
}

func NewVariantKindHandler(ctx context.Context, reqCtx interfaces.RequestContext) (interfaces.KindHandler, apperrors.Error) {
	if reqCtx.Catalog == "" || reqCtx.CatalogID == uuid.Nil {
		return nil, ErrInvalidVariant.Msg("catalog name and ID are required for variant creation")
	}
	return &variantKind{
		req: reqCtx,
	}, nil
}
