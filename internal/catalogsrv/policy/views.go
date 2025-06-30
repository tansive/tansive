package policy

import (
	"context"
	"errors"
	"reflect"
	"strings"

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
	"github.com/tansive/tansive/pkg/types"
)

// viewSchema represents the structure of a view definition
type viewSchema struct {
	ApiVersion string              `json:"apiVersion" validate:"required,validateVersion"`
	Kind       string              `json:"kind" validate:"required,kindValidator"`
	Metadata   interfaces.Metadata `json:"metadata" validate:"required"`
	Spec       viewSpec            `json:"spec" validate:"required"`
}

// viewSpec contains the spec of a view
type viewSpec struct {
	Rules Rules `json:"rules" validate:"required,dive"`
}

// Validate performs validation on the view schema and returns any validation errors.
func (v *viewSchema) Validate() schemaerr.ValidationErrors {
	var validationErrors schemaerr.ValidationErrors
	if v.Kind != catcommon.ViewKind {
		validationErrors = append(validationErrors, schemaerr.ErrUnsupportedKind("kind"))
	}
	err := schemavalidator.V().Struct(v)
	if err == nil {
		// Check for empty rules after struct validation
		if len(v.Spec.Rules) == 0 {
			validationErrors = append(validationErrors, schemaerr.ErrMissingRequiredAttribute("spec.rules"))
		}
		for _, rule := range v.Spec.Rules {
			for _, target := range rule.Targets {
				err := validateResourceURI(string(target))
				if err != nil {
					validationErrors = append(validationErrors, schemaerr.ErrInvalidResourceURI(string(target)+": "+err.Error()))
				}
			}
		}
		return validationErrors
	}

	validatorErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return append(validationErrors, schemaerr.ErrInvalidSchema)
	}

	value := reflect.ValueOf(v).Elem()
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
		case "viewRuleIntentValidator":
			validationErrors = append(validationErrors, schemaerr.ErrInvalidViewRuleIntent(jsonFieldName))
		case "viewRuleActionValidator":
			fieldName, _ := e.Value().(Action)
			validationErrors = append(validationErrors, schemaerr.ErrInvalidViewRuleAction(string(fieldName)))
		default:
			validationErrors = append(validationErrors, schemaerr.ErrValidationFailed(jsonFieldName))
		}
	}

	return validationErrors
}

// parseAndValidateView parses a JSON byte slice into a viewSchema, validates it,
// and optionally overrides the name and catalog fields.
// Returns an error if the JSON is invalid or the schema validation fails.
func parseAndValidateView(ctx context.Context, resourceJSON []byte, m *interfaces.Metadata) (*viewSchema, apperrors.Error) {
	view := &viewSchema{}
	if err := json.Unmarshal(resourceJSON, view); err != nil {
		return nil, ErrInvalidView.Msg("failed to parse view spec")
	}

	if m.Catalog != "" {
		view.Metadata.Catalog = m.Catalog
	}

	if !m.Variant.IsNil() {
		view.Metadata.Variant = m.Variant
	}

	if !m.Namespace.IsNil() {
		view.Metadata.Namespace = m.Namespace
	}

	if err := view.Validate(); err != nil {
		return nil, ErrInvalidSchema.Err(err)
	}

	if err := resolveMetadataIDS(ctx, &view.Metadata); err != nil {
		return nil, err
	}

	return view, nil
}

// resolveMetadataIDS resolves the catalogID and variantID
func resolveMetadataIDS(ctx context.Context, m *interfaces.Metadata) apperrors.Error {
	if m.IDS.CatalogID == uuid.Nil {
		var err apperrors.Error
		m.IDS.CatalogID, err = db.DB(ctx).GetCatalogIDByName(ctx, m.Catalog)
		if err != nil {
			if errors.Is(err, dberror.ErrNotFound) {
				return ErrCatalogNotFound.New("catalog not found: " + m.Catalog)
			}
			log.Ctx(ctx).Error().Err(err).Msg("failed to load catalog")
			return ErrUnableToLoadObject.Msg("unable to load catalog")
		}
	}

	if m.IDS.VariantID == uuid.Nil && !m.Variant.IsNil() {
		var err apperrors.Error
		variant, err := db.DB(ctx).GetVariant(ctx, m.IDS.CatalogID, uuid.Nil, m.Variant.String())
		if err != nil {
			if errors.Is(err, dberror.ErrNotFound) {
				return ErrVariantNotFound.New("variant not found: " + m.Variant.String())
			}
			log.Ctx(ctx).Error().Err(err).Msg("failed to load variant")
			return ErrUnableToLoadObject.Msg("unable to load variant")
		}
		m.IDS.VariantID = variant.VariantID
	}
	return nil
}

// createViewModel creates a view model from a view schema and catalog ID.
// The view definition is bound to the view metadata.  The rules specified are relative to
// the scope of the view definition.
const (
	ViewPurposeCreate = "create"
	ViewPurposeUpdate = "update"
)

func createViewModel(ctx context.Context, view *viewSchema, purpose string) (*models.View, apperrors.Error) {
	viewDef := ViewDefinition{}

	viewDef.Scope.Catalog = view.Metadata.Catalog
	viewDef.Scope.Variant = view.Metadata.Variant.String()
	viewDef.Scope.Namespace = view.Metadata.Namespace.String()
	viewDef.Rules = view.Spec.Rules

	rulesJSON, err := viewDef.ToJSON()
	if err != nil {
		return nil, ErrInvalidView.New("failed to marshal rules: " + err.Error())
	}

	userContext := catcommon.GetUserContext(ctx)
	if userContext == nil || userContext.UserID == "" {
		return nil, dberror.ErrMissingUserContext.Msg("missing user context")
	}
	principal := "user/" + userContext.UserID

	viewModel := &models.View{
		Label:       view.Metadata.Name,
		Description: view.Metadata.Description,
		Info:        nil,
		Rules:       rulesJSON,
		CatalogID:   view.Metadata.IDS.CatalogID,
	}
	if purpose == ViewPurposeCreate {
		viewModel.CreatedBy = principal
	}
	if purpose == ViewPurposeUpdate {
		viewModel.UpdatedBy = principal
	}
	return viewModel, nil
}

// CreateView creates a new view in the database.
func CreateView(ctx context.Context, resourceJSON []byte, m *interfaces.Metadata) (*models.View, apperrors.Error) {
	view, err := parseAndValidateView(ctx, resourceJSON, m)
	if err != nil || view == nil {
		return nil, err
	}

	// Remove duplicates from rules
	view.Spec.Rules = deduplicateRules(view.Spec.Rules)

	v, err := createViewModel(ctx, view, ViewPurposeCreate)
	if err != nil {
		return nil, err
	}

	if err := db.DB(ctx).CreateView(ctx, v); err != nil {
		if errors.Is(err, dberror.ErrAlreadyExists) {
			return nil, ErrAlreadyExists.New("view already exists: " + view.Metadata.Name)
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to create view")
		return nil, ErrViewError.New("failed to create view: " + err.Error())
	}

	return v, nil
}

// UpdateView updates an existing view in the database.
func UpdateView(ctx context.Context, resourceJSON []byte, m *interfaces.Metadata) (*models.View, apperrors.Error) {
	view, err := parseAndValidateView(ctx, resourceJSON, m)
	if err != nil {
		return nil, err
	}

	view.Spec.Rules = deduplicateRules(view.Spec.Rules)

	v, err := createViewModel(ctx, view, ViewPurposeUpdate)
	if err != nil {
		return nil, err
	}

	if err := db.DB(ctx).UpdateView(ctx, v); err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return nil, ErrViewNotFound.New("view not found: " + view.Metadata.Name)
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to update view")
		return nil, ErrViewError.New("failed to update view: " + err.Error())
	}

	return v, nil
}

type viewKind struct {
	reqCtx interfaces.RequestContext
	view   *models.View
}

// Name returns the name of the view resource.
func (v *viewKind) Name() string {
	return v.reqCtx.ObjectName
}

// Location returns the location path of the view resource.
func (v *viewKind) Location() string {
	return "/views/" + v.view.Label
}

// Create creates a new view resource.
func (v *viewKind) Create(ctx context.Context, resourceJSON []byte) (string, apperrors.Error) {
	m := &interfaces.Metadata{}
	m.Catalog = v.reqCtx.Catalog
	if v.reqCtx.Variant != "" {
		m.Variant = types.NullableStringFrom(v.reqCtx.Variant)
	}
	if v.reqCtx.Namespace != "" {
		m.Namespace = types.NullableStringFrom(v.reqCtx.Namespace)
	}
	m.IDS.CatalogID = v.reqCtx.CatalogID
	m.IDS.VariantID = v.reqCtx.VariantID

	view, err := CreateView(ctx, resourceJSON, m)
	if err != nil {
		return "", err
	}
	v.view = view
	return v.Location(), nil
}

// Get retrieves a view resource by its name.
func (v *viewKind) Get(ctx context.Context) ([]byte, apperrors.Error) {
	if v.reqCtx.CatalogID == uuid.Nil || v.reqCtx.ObjectName == "" {
		return nil, ErrInvalidView
	}

	view, err := db.DB(ctx).GetViewByLabel(ctx, v.reqCtx.ObjectName, v.reqCtx.CatalogID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return nil, ErrViewNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to load view")
		return nil, ErrUnableToLoadObject.Msg("unable to load view")
	}

	v.view = view

	// Convert the view model to JSON
	viewSchema := &viewSchema{
		ApiVersion: catcommon.ApiVersion,
		Kind:       catcommon.ViewKind,
		Metadata: interfaces.Metadata{
			Name:        view.Label,
			Description: view.Description,
		},
	}

	// Parse the rules from the view model
	var viewDef ViewDefinition
	if err := json.Unmarshal(view.Rules, &viewDef); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal view rules")
		return nil, ErrUnableToLoadObject.Msg("unable to unmarshal view rules")
	}

	viewSchema.Spec.Rules = viewDef.Rules

	if viewDef.Scope.Catalog != v.reqCtx.Catalog {
		return nil, ErrInvalidView.New("view catalog does not match request catalog")
	}

	viewSchema.Metadata.Catalog = viewDef.Scope.Catalog
	if viewDef.Scope.Variant != "" {
		viewSchema.Metadata.Variant = types.NullableStringFrom(viewDef.Scope.Variant)
	}
	if viewDef.Scope.Namespace != "" {
		viewSchema.Metadata.Namespace = types.NullableStringFrom(viewDef.Scope.Namespace)
	}

	jsonData, e := json.Marshal(viewSchema)
	if e != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to marshal view schema")
		return nil, ErrUnableToLoadObject.Msg("unable to fetch view schema")
	}

	return jsonData, nil
}

// Delete removes a view resource.
func (v *viewKind) Delete(ctx context.Context) apperrors.Error {
	if v.reqCtx.CatalogID == uuid.Nil || v.reqCtx.ObjectName == "" {
		return ErrInvalidView
	}

	err := db.DB(ctx).DeleteViewByLabel(ctx, v.reqCtx.ObjectName, v.reqCtx.CatalogID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return nil
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to delete view")
		return ErrUnableToDeleteObject.Msg("unable to delete view")
	}

	return nil
}

// Update modifies an existing view resource.
func (v *viewKind) Update(ctx context.Context, resourceJSON []byte) apperrors.Error {
	m := &interfaces.Metadata{}
	m.Catalog = v.reqCtx.Catalog
	if v.reqCtx.Variant != "" {
		m.Variant = types.NullableStringFrom(v.reqCtx.Variant)
	}
	if v.reqCtx.Namespace != "" {
		m.Namespace = types.NullableStringFrom(v.reqCtx.Namespace)
	}
	m.IDS.CatalogID = v.reqCtx.CatalogID
	m.IDS.VariantID = v.reqCtx.VariantID

	_, err := UpdateView(ctx, resourceJSON, m)
	if err != nil {
		return err
	}

	return nil
}

func (v *viewKind) List(ctx context.Context) ([]byte, apperrors.Error) {
	if v.reqCtx.CatalogID == uuid.Nil {
		return nil, ErrInvalidCatalog
	}

	views, err := db.DB(ctx).ListViewsByCatalog(ctx, v.reqCtx.CatalogID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to load views")
		return nil, ErrUnableToLoadObject.Msg("unable to load view")
	}

	type viewItem struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	type viewListRsp struct {
		Views []viewItem `json:"views"`
	}

	viewsRsp := viewListRsp{
		Views: []viewItem{},
	}
	for _, view := range views {
		if strings.HasPrefix(view.Label, "_") {
			continue
		}
		viewsRsp.Views = append(viewsRsp.Views, viewItem{
			Name:        view.Label,
			Description: view.Description,
		})
	}

	jsonData, e := json.Marshal(viewsRsp)
	if e != nil {
		log.Ctx(ctx).Error().Err(e).Msg("failed to marshal view list")
		return nil, ErrUnableToLoadObject.Msg("unable to marshal view list")
	}

	return jsonData, nil
}

// NewViewKindHandler creates a new view resource manager.
func NewViewKindHandler(ctx context.Context, reqCtx interfaces.RequestContext) (interfaces.KindHandler, apperrors.Error) {
	if reqCtx.Catalog == "" || reqCtx.CatalogID == uuid.Nil {
		return nil, ErrInvalidCatalog
	}
	return &viewKind{
		reqCtx: reqCtx,
	}, nil
}
