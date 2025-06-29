package catalogmanager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"

	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/objectstore"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/pkg/api"
	"github.com/tansive/tansive/pkg/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// SkillSetManager defines the interface for managing a single skillset.
type SkillSetManager interface {
	Metadata() interfaces.Metadata
	FullyQualifiedName() string
	Save(ctx context.Context) apperrors.Error
	JSON(ctx context.Context) ([]byte, apperrors.Error)
	SpecJSON(ctx context.Context) ([]byte, apperrors.Error)
	GetStoragePath() string
	StorageRepresentation() *objectstore.ObjectStorageRepresentation
	GetSkillMetadata() (SkillMetadata, apperrors.Error)
	GetResourcePath() string
	GetSourceForSkill(skillName string) (SkillSetSource, apperrors.Error)
	GetSourceByName(sourceName string) (SkillSetSource, apperrors.Error)
	GetSkill(name string) (Skill, apperrors.Error)
	GetAllSkills() []Skill
	GetAllSkillsAsLLMTools(viewDef *policy.ViewDefinition) []api.LLMTool
	GetContext(name string) (SkillSetContext, apperrors.Error)
	GetContextValue(name string) (types.NullableAny, apperrors.Error)
	SetContextValue(name string, value types.NullableAny) apperrors.Error
	GetRunnerTypes() []catcommon.RunnerID
	ValidateInputForSkill(ctx context.Context, skillName string, input map[string]any) apperrors.Error
}

// NewSkillSetManager creates a new Sk sillSetManager instance from the pro vided JSON schema and metadata.
func NewSkillSetManager(ctx context.Context, rsrcJSON []byte, m *interfaces.Metadata) (SkillSetManager, apperrors.Error) {
	if len(rsrcJSON) == 0 {
		return nil, ErrEmptySchema
	}

	// Get the metadata, replace fields in JSON from provided metadata, and set defaults.
	rsrcJSON, m, err := canonicalizeMetadata(rsrcJSON, catcommon.SkillSetKind, m)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to canonicalize metadata")
		return nil, ErrSchemaSerialization
	}

	var skillset SkillSet
	if err := json.Unmarshal(rsrcJSON, &skillset); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal skillset")
		return nil, ErrSchemaValidation
	}

	if validationErrs := skillset.Validate(); validationErrs != nil {
		log.Ctx(ctx).Error().Err(validationErrs).Msg("Skillset validation failed")
		return nil, ErrSchemaValidation.Msg(validationErrs.Error())
	}

	skillset.Metadata = *m

	return &skillSetManager{skillSet: skillset}, nil
}

// GetSkillSetManager gets a skillset manager given a skillset path.
func GetSkillSetManager(ctx context.Context, skillSetPath string, viewScope ...policy.Scope) (SkillSetManager, apperrors.Error) {
	if skillSetPath == "" {
		return nil, ErrInvalidObject.Msg("skillset path is required")
	}

	var m *interfaces.Metadata
	if len(viewScope) > 0 {
		m = &interfaces.Metadata{
			Catalog:   viewScope[0].Catalog,
			Variant:   types.NullableStringFrom(viewScope[0].Variant),
			Namespace: types.NullableStringFrom(viewScope[0].Namespace),
		}
	} else {
		m = &interfaces.Metadata{
			Catalog: catcommon.GetCatalog(ctx),
		}
		if v := catcommon.GetVariant(ctx); v != "" {
			m.Variant = types.NullableStringFrom(v)
		}
		if n := catcommon.GetNamespace(ctx); n != "" {
			m.Namespace = types.NullableStringFrom(n)
		}
	}

	skillSetName := path.Base(skillSetPath)
	if skillSetName == "" {
		return nil, ErrInvalidObject.Msg("skillset name is required")
	}
	skillSetPath = path.Dir(skillSetPath)
	m.Name = skillSetName
	m.Path = skillSetPath

	skillSetManager, err := LoadSkillSetManagerByPath(ctx, m)
	if err != nil {
		return nil, err
	}
	return skillSetManager, nil
}

// LoadSkillSetManagerByPath loads a skillset manager from the database by path.
func LoadSkillSetManagerByPath(ctx context.Context, m *interfaces.Metadata) (SkillSetManager, apperrors.Error) {
	if m == nil {
		return nil, ErrInvalidObject.Msg("unable to infer object metadata")
	}

	// Get the directory ID for the skillset
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

	pathWithName := path.Clean(m.GetStoragePath(catcommon.CatalogObjectTypeSkillset) + "/" + m.Name)

	obj, err := db.DB(ctx).GetSkillSetObject(ctx, pathWithName, variant.SkillsetDirectoryID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return nil, ErrObjectNotFound.Msg("skillset not found")
		}
		return nil, err
	}

	return skillSetManagerFromObject(ctx, obj, m)
}

// LoadSkillSetManagerByHash loads a skillset manager from the database by hash.
func LoadSkillSetManagerByHash(ctx context.Context, hash string, m *interfaces.Metadata) (SkillSetManager, apperrors.Error) {
	// get the object from catalog object store
	obj, err := db.DB(ctx).GetCatalogObject(ctx, hash)
	if err != nil {
		return nil, err
	}
	return skillSetManagerFromObject(ctx, obj, m)
}

func skillSetManagerFromObject(ctx context.Context, obj *models.CatalogObject, m *interfaces.Metadata) (SkillSetManager, apperrors.Error) {
	if obj == nil {
		return nil, ErrEmptySchema
	}

	var storageRep objectstore.ObjectStorageRepresentation
	if err := json.Unmarshal(obj.Data, &storageRep); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal skillset")
		return nil, ErrSchemaValidation
	}

	if storageRep.Type != catcommon.CatalogObjectTypeSkillset {
		log.Ctx(ctx).Error().Msg("Invalid type")
		return nil, ErrUnableToLoadObject
	}

	sm := &skillSetManager{}
	if err := json.Unmarshal(storageRep.Spec, &sm.skillSet.Spec); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal skillset schema spec")
		return nil, ErrUnableToLoadObject
	}

	sm.skillSet.Kind = catcommon.SkillSetKind
	sm.skillSet.ApiVersion = storageRep.Version
	sm.skillSet.Metadata = *m
	sm.skillSet.Metadata.Description = storageRep.Description

	return sm, nil
}

var _ interfaces.KindHandler = &skillsetKindHandler{}

// skillsetKindHandler implements the KindHandler interface for managing individual skillsets.
// It handles CRUD operations for skillsets and maintains the request context.
type skillsetKindHandler struct {
	req interfaces.RequestContext
	sm  SkillSetManager
}

// Name returns the name of the skillset from the request context.
func (h *skillsetKindHandler) Name() string {
	return h.req.ObjectName
}

// Location returns the fully qualified path to the skillset, including any query parameters.
// The path is constructed using the skillset name and namespace (if present).
func (h *skillsetKindHandler) Location() string {
	objName := catcommon.KindNameFromObjectType(h.req.ObjectType)
	loc := path.Clean("/" + objName + h.sm.FullyQualifiedName())

	q := url.Values{}
	if namespace := h.sm.Metadata().Namespace.String(); namespace != "" {
		q.Set("namespace", namespace)
	}

	if qStr := q.Encode(); qStr != "" {
		loc += "?" + qStr
	}

	return loc
}

// Manager returns the underlying SkillSetManager instance.
func (h *skillsetKindHandler) Manager() SkillSetManager {
	return h.sm
}

// Create creates a new skillset from the provided JSON data.
// It validates the input, saves the skillset, and updates the request context with the new skillset's metadata.
func (h *skillsetKindHandler) Create(ctx context.Context, skillsetJSON []byte) (string, apperrors.Error) {
	m := &interfaces.Metadata{
		Catalog:   h.req.Catalog,
		Variant:   types.NullableStringFrom(h.req.Variant),
		Namespace: types.NullableStringFrom(h.req.Namespace),
	}

	sm, err := NewSkillSetManager(ctx, skillsetJSON, m)
	if err != nil {
		return "", err
	}

	if err := sm.Save(ctx); err != nil {
		return "", err
	}

	h.req.ObjectName = sm.Metadata().Name
	h.req.ObjectPath = sm.Metadata().Path
	h.req.ObjectType = catcommon.CatalogObjectTypeSkillset
	h.sm = sm

	// Update request context with metadata if not set
	if h.req.Catalog == "" {
		h.req.Catalog = sm.Metadata().Catalog
	}
	if h.req.Variant == "" {
		h.req.Variant = sm.Metadata().Variant.String()
	}
	if h.req.Namespace == "" {
		h.req.Namespace = sm.Metadata().Namespace.String()
	}

	return h.Location(), nil
}

// Get retrieves a skillset by its path and returns it as JSON.
// It validates the metadata and loads the skillset from storage.
// Hidden context values are replaced with their SHA256 hash (8 characters).
func (h *skillsetKindHandler) Get(ctx context.Context) ([]byte, apperrors.Error) {
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

	sm, err := LoadSkillSetManagerByPath(ctx, m)
	if err != nil {
		return nil, err
	}

	// Get the JSON representation
	jsonData, err := sm.JSON(ctx)
	if err != nil {
		return nil, err
	}

	// Process hidden context values
	jsonData, err = h.hashHiddenContextValues(jsonData)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to hash hidden context values")
		return nil, ErrUnableToLoadObject.Msg("failed to process context values")
	}

	return jsonData, nil
}

// hashHiddenContextValues processes the JSON to find context values with hidden=true
// and replaces their values with SHA256 hashes (8 characters)
func (h *skillsetKindHandler) hashHiddenContextValues(jsonData []byte) ([]byte, apperrors.Error) {
	contexts := gjson.GetBytes(jsonData, "spec.context")
	if !contexts.IsArray() {
		return jsonData, nil
	}

	for i, context := range contexts.Array() {
		hidden := gjson.Get(context.Raw, "attributes.hidden")
		if !hidden.Bool() {
			continue
		}

		value := gjson.Get(context.Raw, "value")
		if !value.Exists() || value.Type == gjson.Null {
			continue
		}

		valueBytes := []byte(value.Raw)
		hash := sha256.Sum256(valueBytes)
		hashHex := hex.EncodeToString(hash[:])[:8] // Take first 8 characters

		path := fmt.Sprintf("spec.context.%d.value", i)
		var err error
		jsonData, err = sjson.SetBytes(jsonData, path, hashHex)
		if err != nil {
			return nil, ErrUnableToLoadObject.Msg("failed to replace context value with hash")
		}
	}

	return jsonData, nil
}

// Update updates an existing skillset with new data.
// It validates the input, checks for the skillset's existence, and saves the changes.
func (h *skillsetKindHandler) Update(ctx context.Context, skillsetJSON []byte) apperrors.Error {
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
	existing, err := LoadSkillSetManagerByPath(ctx, m)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrObjectNotFound
	}

	sm, err := NewSkillSetManager(ctx, skillsetJSON, m)
	if err != nil {
		return err
	}
	return sm.Save(ctx)
}

// Delete removes a skillset from storage.
// It validates the metadata and deletes the skillset if it exists.
func (h *skillsetKindHandler) Delete(ctx context.Context) apperrors.Error {
	m := &interfaces.Metadata{
		Catalog:   h.req.Catalog,
		Variant:   types.NullableStringFrom(h.req.Variant),
		Path:      h.req.ObjectPath,
		Name:      h.req.ObjectName,
		Namespace: types.NullableStringFrom(h.req.Namespace),
	}

	if err := DeleteSkillSet(ctx, m); err != nil {
		pathWithName := path.Clean(m.GetStoragePath(h.req.ObjectType) + "/" + h.req.ObjectName)
		log.Ctx(ctx).Error().Err(err).Str("path", pathWithName).Msg("Failed to delete object")
		return err
	}
	return nil
}

func (h *skillsetKindHandler) List(ctx context.Context) ([]byte, apperrors.Error) {
	variant, err := db.DB(ctx).GetVariantByID(ctx, h.req.VariantID)
	if err != nil {
		return nil, ErrInvalidVariant
	}

	skillsets, err := db.DB(ctx).ListSkillSets(ctx, variant.SkillsetDirectoryID)
	if err != nil {
		return nil, ErrCatalogError.Msg("unable to list skillsets")
	}

	skillsetList := make(map[string]json.RawMessage)
	for _, skillset := range skillsets {
		m := &interfaces.Metadata{
			Catalog:   h.req.Catalog,
			Variant:   types.NullableStringFrom(h.req.Variant),
			Namespace: types.NullableStringFrom(h.req.Namespace),
		}
		m.SetNameAndPathFromStoragePath(skillset.Path)
		sm, err := LoadSkillSetManagerByHash(ctx, skillset.Hash, m)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("path", skillset.Path).Msg("Failed to load skillset")
			continue
		}

		j, err := sm.JSON(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("path", skillset.Path).Msg("Failed to marshal skillset")
			continue
		}
		skillsetList[path.Clean(m.Path+"/"+m.Name)] = j
	}

	j, goErr := json.Marshal(skillsetList)
	if goErr != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to marshal skillset list")
		return nil, ErrInvalidSkillSetDefinition
	}

	return j, nil
}

func NewSkillSetKindHandler(ctx context.Context, req interfaces.RequestContext) (interfaces.KindHandler, apperrors.Error) {
	if req.Catalog == "" {
		return nil, ErrInvalidCatalog
	}
	if req.Variant == "" {
		return nil, ErrInvalidVariant
	}
	return &skillsetKindHandler{
		req: req,
	}, nil
}

// This is a helper function without validation, and to be used from client code.
func SkillSetManagerFromJSON(ctx context.Context, jsonData []byte) (SkillSetManager, apperrors.Error) {
	sm := &skillSetManager{}
	if err := json.Unmarshal(jsonData, &sm.skillSet); err != nil {
		return nil, ErrSchemaValidation.Msg(err.Error())
	}
	return sm, nil
}
