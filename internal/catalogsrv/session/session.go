package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"reflect"
	"time"

	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	schemaerr "github.com/tansive/tansive/internal/catalogsrv/schema/errors"
	"github.com/tansive/tansive/internal/catalogsrv/schema/schemavalidator"
	"github.com/tansive/tansive/internal/catalogsrv/tangent"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tidwall/gjson"
)

// SessionSpec defines the structure for session creation requests
type SessionSpec struct {
	SkillPath        string          `json:"skillPath" validate:"required,skillPathValidator"`
	ViewName         string          `json:"viewName" validate:"required,resourceNameValidator"`
	SessionVariables json.RawMessage `json:"sessionVariables" validate:"omitempty"`
	InputArgs        json.RawMessage `json:"inputArgs" validate:"omitempty"`
}

// variableSchema defines the JSON schema for session variables
const variableSchema = `
{
  "type": "object",
  "maxProperties": %d,
  "propertyNames": {
    "pattern": "^[a-zA-Z0-9._-]+$"
  },
  "additionalProperties": {
    "type": ["string", "number", "boolean", "object", "array", "null"]
  }
}`

type SessionInfo struct {
	SessionVariables map[string]any         `json:"sessionVariables" validate:"omitempty"`
	InputArgs        map[string]any         `json:"inputArgs" validate:"omitempty"`
	ViewDefinition   *policy.ViewDefinition `json:"viewDefinition" validate:"omitempty"`
	Interactive      bool                   `json:"interactive" validate:"omitempty"`
	CodeChallenge    string                 `json:"codeChallenge" validate:"omitempty"`
}

var variableSchemaCompiled *jsonschema.Schema

// sessionManager implements the SessionManager interface
type sessionManager struct {
	session         *models.Session
	skillSetManager catalogmanager.SkillSetManager
	viewManager     policy.ViewManager
}

func Init() {
	schema := fmt.Sprintf(variableSchema, config.Config().Session.MaxVariables)
	compiledSchema, err := compileSchema(schema)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to compile session variables schema")
	}
	variableSchemaCompiled = compiledSchema
}

type requestOptions struct {
	codeChallenge string
	interactive   bool
}

type RequestOptions func(o *requestOptions)

func WithCodeChallenge(codeChallenge string) RequestOptions {
	return func(o *requestOptions) {
		o.codeChallenge = codeChallenge
	}
}

func WithInteractive(interactive bool) RequestOptions {
	return func(o *requestOptions) {
		o.interactive = interactive
	}
}

// NewSession creates a new session with the given specification. It validates the session spec,
// checks permissions, and initializes the session with the provided configuration.
// The function requires valid catalog ID, variant ID, and user ID in the context.
// Returns a SessionManager interface and any error that occurred during creation.
func NewSession(ctx context.Context, rsrcSpec []byte, opts ...RequestOptions) (SessionManager, *tangent.Tangent, apperrors.Error) {
	// Validate required IDs first
	if err := validateRequiredIDs(ctx); err != nil {
		return nil, nil, err
	}

	requestOptions := &requestOptions{}
	for _, opt := range opts {
		opt(requestOptions)
	}

	sessionSpec, err := resolveSessionSpec(rsrcSpec)
	if err != nil {
		return nil, nil, err
	}

	// Validate session specification
	if err := validateSessionSpec(ctx, sessionSpec); err != nil {
		return nil, nil, err
	}

	// Parse input arguments and session variables
	inputArgs, sessionVariables, err := parseSessionData(sessionSpec)
	if err != nil {
		return nil, nil, err
	}

	// Resolve view and skill set managers
	viewManager, skillSetManager, skillObj, err := resolveManagersAndSkill(ctx, sessionSpec)
	if err != nil {
		return nil, nil, err
	}

	// Validate skill input and permissions
	if err := validateSkillAndPermissions(ctx, skillObj, viewManager, skillSetManager, inputArgs); err != nil {
		return nil, nil, err
	}

	// Create session info
	sessionInfo, err := createSessionInfo(sessionSpec, inputArgs, sessionVariables, viewManager, requestOptions)
	if err != nil {
		return nil, nil, err
	}

	// Get Tangent
	tangent, err := tangent.GetTangentWithCapabilities(ctx, skillSetManager.GetRunnerTypes())
	if err != nil {
		return nil, nil, err
	}

	// Create session object
	session, err := createSessionObject(ctx, sessionSpec, sessionInfo, viewManager, tangent)
	if err != nil {
		return nil, nil, err
	}

	return &sessionManager{
		session:         session,
		skillSetManager: skillSetManager,
		viewManager:     viewManager,
	}, tangent, nil
}

// validateRequiredIDs validates that required IDs are present in the context
func validateRequiredIDs(ctx context.Context) apperrors.Error {
	catalogID := catcommon.GetCatalogID(ctx)
	if catalogID == uuid.Nil {
		return ErrInvalidObject.Msg("unable to determine catalog")
	}

	userID := catcommon.GetUserID(ctx)
	if userID == "" {
		return ErrInvalidObject.Msg("user ID is required")
	}

	return nil
}

// validateSessionSpec validates the session specification
func validateSessionSpec(ctx context.Context, sessionSpec SessionSpec) apperrors.Error {
	// validate SkillSet Use policy
	if err := validateSkillSetUsePolicy(ctx, sessionSpec.SkillPath); err != nil {
		return err
	}

	skill := path.Base(sessionSpec.SkillPath)
	skillSetPath := path.Dir(sessionSpec.SkillPath)
	if skill == "" || skillSetPath == "" {
		return ErrInvalidObject.Msg("invalid skill path")
	}

	// Validate view policy
	if err := validateViewPolicy(ctx, sessionSpec.ViewName); err != nil {
		return err
	}

	return nil
}

// parseSessionData parses input arguments and session variables from the session specification
func parseSessionData(sessionSpec SessionSpec) (map[string]any, map[string]any, apperrors.Error) {
	inputArgs := make(map[string]any)
	if len(sessionSpec.InputArgs) > 0 {
		if err := json.Unmarshal(sessionSpec.InputArgs, &inputArgs); err != nil {
			return nil, nil, ErrInvalidObject.Msg("failed to unmarshal input args: " + err.Error())
		}
	}

	sessionVariables := make(map[string]any)
	if len(sessionSpec.SessionVariables) > 0 {
		if err := json.Unmarshal(sessionSpec.SessionVariables, &sessionVariables); err != nil {
			return nil, nil, ErrInvalidObject.Msg("failed to unmarshal session variables: " + err.Error())
		}
	}

	return inputArgs, sessionVariables, nil
}

// resolveManagersAndSkill resolves view manager, skill set manager, and skill object
func resolveManagersAndSkill(ctx context.Context, sessionSpec SessionSpec) (policy.ViewManager, catalogmanager.SkillSetManager, catalogmanager.Skill, apperrors.Error) {
	viewManager, err := resolveViewByLabel(ctx, sessionSpec.ViewName)
	if err != nil {
		return nil, nil, catalogmanager.Skill{}, err
	}

	skillSetPath := path.Dir(sessionSpec.SkillPath)
	skillSetManager, err := resolveSkillSetManager(ctx, skillSetPath, viewManager.Scope())
	if err != nil {
		return nil, nil, catalogmanager.Skill{}, err
	}

	skill := path.Base(sessionSpec.SkillPath)
	skillObj, err := skillSetManager.GetSkill(skill)
	if err != nil {
		return nil, nil, catalogmanager.Skill{}, err
	}

	return viewManager, skillSetManager, skillObj, nil
}

// validateSkillAndPermissions validates skill input and action permissions
func validateSkillAndPermissions(ctx context.Context, skillObj catalogmanager.Skill, viewManager policy.ViewManager, skillSetManager catalogmanager.SkillSetManager, inputArgs map[string]any) apperrors.Error {
	_ = ctx

	// Validate skill input
	err := skillObj.ValidateInput(inputArgs)
	if err != nil {
		return err
	}

	// Validate action permissions
	exportedActions := skillObj.GetExportedActions()
	viewDef := viewManager.GetViewDefinition()
	allowed, _, err := policy.AreActionsAllowedOnResource(viewDef, skillSetManager.GetResourcePath(), exportedActions)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrDisallowedByPolicy.Msg("use of skill is blocked by policy")
	}

	return nil
}

// createSessionInfo creates the session info object
func createSessionInfo(_ SessionSpec, inputArgs map[string]any, sessionVariables map[string]any, viewManager policy.ViewManager, requestOptions *requestOptions) ([]byte, apperrors.Error) {
	viewDef := viewManager.GetViewDefinition()
	sessionInfo := SessionInfo{
		SessionVariables: sessionVariables,
		InputArgs:        inputArgs,
		ViewDefinition:   viewDef,
		Interactive:      requestOptions.interactive,
		CodeChallenge:    requestOptions.codeChallenge,
	}
	sessionInfoJSON, goerr := json.Marshal(sessionInfo)
	if goerr != nil {
		return nil, ErrInvalidObject.Msg("failed to marshal session info: " + goerr.Error())
	}
	return sessionInfoJSON, nil
}

// createSessionObject creates the session object
func createSessionObject(ctx context.Context, sessionSpec SessionSpec, sessionInfo []byte, viewManager policy.ViewManager, tangent *tangent.Tangent) (*models.Session, apperrors.Error) {
	catalogID := catcommon.GetCatalogID(ctx)
	userID := catcommon.GetUserID(ctx)
	variantID := catcommon.GetVariantID(ctx)

	skill := path.Base(sessionSpec.SkillPath)
	skillSetPath := path.Dir(sessionSpec.SkillPath)

	sessionID := uuid.New()
	session := &models.Session{
		SessionID:     sessionID,
		SkillSet:      skillSetPath,
		Skill:         skill,
		ViewID:        viewManager.ID(),
		TangentID:     tangent.ID,
		StatusSummary: string(SessionStatusCreated),
		Status:        nil,
		Info:          sessionInfo,
		UserID:        userID,
		CatalogID:     catalogID,
		VariantID:     variantID,
		StartedAt:     time.Now(),
		EndedAt:       time.Time{},
		ExpiresAt:     time.Now().Add(config.Config().Session.GetExpirationTimeOrDefault()),
	}

	return session, nil
}

// Save persists the session to the database.
// Returns an error if the save operation fails.
func (s *sessionManager) Save(ctx context.Context) apperrors.Error {
	err := db.DB(ctx).UpsertSession(ctx, s.session)
	if err != nil {
		return ErrInvalidObject.Msg("failed to save session: " + err.Error())
	}
	return nil
}

// GetSession retrieves an existing session by its ID.
// Returns a SessionManager interface and any error that occurred during retrieval.
func GetSession(ctx context.Context, sessionID uuid.UUID) (SessionManager, apperrors.Error) {
	log.Ctx(ctx).Info().Msgf("Getting session %s", sessionID.String())
	session, err := db.DB(ctx).GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrInvalidObject.Msg("failed to get session: " + err.Error())
	}
	viewManager, err := resolveViewByID(ctx, session.ViewID)
	if err != nil {
		return nil, err
	}
	skillSetManager, err := resolveSkillSetManager(ctx, session.SkillSet, viewManager.Scope())
	if err != nil {
		return nil, err
	}
	sm := &sessionManager{
		session:         session,
		skillSetManager: skillSetManager,
		viewManager:     viewManager,
	}
	return sm, nil
}

// resolveSessionSpec parses and validates the session specification
func resolveSessionSpec(rsrcSpec []byte) (SessionSpec, apperrors.Error) {
	sessionSpec := SessionSpec{}
	if err := json.Unmarshal(rsrcSpec, &sessionSpec); err != nil {
		return sessionSpec, ErrInvalidSession.Msg("invalid session spec: " + err.Error())
	}

	// Validate the session spec
	validationErrors := sessionSpec.Validate()
	if len(validationErrors) > 0 {
		return sessionSpec, ErrInvalidSession.Msg(validationErrors.Error())
	}

	return sessionSpec, nil
}

// validateSessionVariables validates the session variables against the schema
func validateSessionVariables(variables json.RawMessage) schemaerr.ValidationErrors {
	if len(variables) == 0 {
		return nil
	}

	var parsed any
	if err := json.Unmarshal(variables, &parsed); err != nil {
		return schemaerr.ValidationErrors{schemaerr.ErrValidationFailed("invalid session variables: " + err.Error())}
	}

	if err := variableSchemaCompiled.Validate(parsed); err != nil {
		msg := fmt.Sprintf("session variables must be key-value json objects with max %d properties: %v", config.Config().Session.MaxVariables, err)
		return schemaerr.ValidationErrors{schemaerr.ErrValidationFailed(msg)}
	}

	return nil
}

// validateSessionSpecFields validates the session spec fields using the validator
func validateSessionSpecFields(s *SessionSpec) schemaerr.ValidationErrors {
	err := schemavalidator.V().Struct(s)
	if err == nil {
		return nil
	}

	validatorErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return schemaerr.ValidationErrors{schemaerr.ErrInvalidSchema}
	}

	value := reflect.ValueOf(s).Elem()
	typeOfCS := value.Type()
	var validationErrors schemaerr.ValidationErrors

	for _, e := range validatorErrors {
		jsonFieldName := schemavalidator.GetJSONFieldPath(value, typeOfCS, e.StructField())
		switch e.Tag() {
		case "required":
			validationErrors = append(validationErrors, schemaerr.ErrMissingRequiredAttribute(jsonFieldName))
		case "resourceNameValidator":
			val, _ := e.Value().(string)
			validationErrors = append(validationErrors, schemaerr.ErrInvalidNameFormat(jsonFieldName, val))
		case "skillPathValidator":
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

// Validate validates the session specification against required rules and constraints.
// Checks both the struct fields and session variables for validity.
// Returns a collection of validation errors if any are found.
func (s *SessionSpec) Validate() schemaerr.ValidationErrors {
	var validationErrors schemaerr.ValidationErrors

	// Validate struct fields
	if errs := validateSessionSpecFields(s); len(errs) > 0 {
		validationErrors = append(validationErrors, errs...)
	}

	// Validate variables
	if errs := validateSessionVariables(s.SessionVariables); len(errs) > 0 {
		validationErrors = append(validationErrors, errs...)
	}

	return validationErrors
}

// resolveViewByLabel creates a new view manager for the given view name
func resolveViewByLabel(ctx context.Context, viewName string) (policy.ViewManager, apperrors.Error) {
	viewManager, err := policy.NewViewManagerByViewLabel(ctx, viewName)
	if err != nil {
		return nil, err
	}
	return viewManager, nil
}

// resolveViewByID creates a new view manager for the given view ID
func resolveViewByID(ctx context.Context, viewID uuid.UUID) (policy.ViewManager, apperrors.Error) {
	viewManager, err := policy.NewViewManagerByViewID(ctx, viewID)
	if err != nil {
		return nil, err
	}
	return viewManager, nil
}

// resolveSkillSetManager creates a new skill set manager for the given path
func resolveSkillSetManager(ctx context.Context, skillSetPath string, viewScope policy.Scope) (catalogmanager.SkillSetManager, apperrors.Error) {
	if skillSetPath == "" {
		return nil, ErrInvalidObject.Msg("skillset path is required")
	}

	skillSetManager, err := catalogmanager.GetSkillSetManager(ctx, skillSetPath, viewScope)
	if err != nil {
		return nil, err
	}
	return skillSetManager, nil
}

// compileSchema compiles a JSON schema string into a jsonschema.Schema
func compileSchema(schema string) (*jsonschema.Schema, error) {
	if !gjson.Valid(schema) {
		return nil, fmt.Errorf("invalid JSON schema")
	}

	compiler := jsonschema.NewCompiler()
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

func validateViewPolicy(ctx context.Context, view string) apperrors.Error {
	if view == "" {
		return ErrInvalidObject.Msg("view is required")
	}

	allowed, err := policy.CanAdoptView(ctx, view)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrDisallowedByPolicy.Msg("view is not allowed to be adopted")
	}

	return nil
}

func validateSkillSetUsePolicy(ctx context.Context, skillSetPath string) apperrors.Error {
	if skillSetPath == "" {
		return ErrInvalidObject.Msg("skillset path is required")
	}

	allowed, err := policy.CanUseSkillSet(ctx, path.Dir(skillSetPath))
	if err != nil {
		return err
	}
	if !allowed {
		return ErrDisallowedByPolicy.Msg("skillset is not allowed to be used")
	}
	return nil
}
