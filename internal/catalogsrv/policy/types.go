package policy

import (
	"encoding/json"

	"github.com/tansive/tansive/internal/common/httpx"
)

type Intent string

const (
	IntentAllow Intent = "Allow"
	IntentDeny  Intent = "Deny"
)

type Action string

const (
	ActionAllow             Action = "allow"
	ActionCatalogAdmin      Action = "system.catalog.admin"
	ActionCatalogList       Action = "system.catalog.list"
	ActionCatalogAdoptView  Action = "system.catalog.adoptView"
	ActionCatalogCreateView Action = "system.catalog.createView"
	ActionViewAdmin         Action = "system.view.admin"
	ActionVariantAdmin      Action = "system.variant.admin"
	ActionVariantClone      Action = "system.variant.clone"
	ActionVariantList       Action = "system.variant.list"
	ActionNamespaceCreate   Action = "system.namespace.create"
	ActionNamespaceList     Action = "system.namespace.list"
	ActionNamespaceAdmin    Action = "system.namespace.admin"
	ActionResourceCreate    Action = "system.resource.create"
	ActionResourceRead      Action = "system.resource.read"
	ActionResourceEdit      Action = "system.resource.edit"
	ActionResourceDelete    Action = "system.resource.delete"
	ActionResourceGet       Action = "system.resource.get"
	ActionResourcePut       Action = "system.resource.put"
	ActionResourceList      Action = "system.resource.list"
	ActionSkillSetAdmin     Action = "system.skillset.admin"
	ActionSkillSetCreate    Action = "system.skillset.create"
	ActionSkillSetRead      Action = "system.skillset.read"
	ActionSkillSetEdit      Action = "system.skillset.edit"
	ActionSkillSetDelete    Action = "system.skillset.delete"
	ActionSkillSetList      Action = "system.skillset.list"
	ActionSkillSetUse       Action = "system.skillset.use"
	ActionTangentCreate     Action = "system.tangent.create"
	ActionTangentDelete     Action = "system.tangent.delete"
)

var ValidActions = []Action{
	ActionCatalogAdmin,
	ActionCatalogList,
	ActionCatalogAdoptView,
	ActionCatalogCreateView,
	ActionVariantAdmin,
	ActionVariantClone,
	ActionVariantList,
	ActionNamespaceCreate,
	ActionNamespaceList,
	ActionNamespaceAdmin,
	ActionResourceCreate,
	ActionResourceRead,
	ActionResourceEdit,
	ActionResourceDelete,
	ActionResourceGet,
	ActionResourcePut,
	ActionResourceList,
	ActionSkillSetCreate,
	ActionSkillSetRead,
	ActionSkillSetEdit,
	ActionSkillSetDelete,
	ActionSkillSetList,
	ActionSkillSetUse,
}

type Rule struct {
	Intent  Intent           `json:"intent" validate:"required,viewRuleIntentValidator"`
	Actions []Action         `json:"actions" validate:"required,dive,viewRuleActionValidator"`
	Targets []TargetResource `json:"targets" validate:"-"`
}

type TargetResource string
type Rules []Rule
type Scope struct {
	Catalog   string `json:"catalog"`
	Variant   string `json:"variant"`
	Namespace string `json:"namespace"`
}

func (v Scope) Equals(other Scope) bool {
	return v.Catalog == other.Catalog &&
		v.Variant == other.Variant &&
		v.Namespace == other.Namespace
}

type ViewDefinition struct {
	Scope Scope `json:"scope" validate:"required"`
	Rules Rules `json:"rules" validate:"required,dive"`
}

func (v ViewDefinition) DeepCopy() ViewDefinition {
	return ViewDefinition{
		Scope: v.Scope, // Scope is a struct of strings (safe to copy)
		Rules: v.Rules.DeepCopy(),
	}
}

func (r Rules) DeepCopy() Rules {
	copied := make(Rules, len(r))
	for i, rule := range r {
		copied[i] = rule.DeepCopy()
	}
	return copied
}

func (r Rule) DeepCopy() Rule {
	actionsCopy := make([]Action, len(r.Actions))
	copy(actionsCopy, r.Actions)

	targetsCopy := make([]TargetResource, len(r.Targets))
	copy(targetsCopy, r.Targets)

	return Rule{
		Intent:  r.Intent,
		Actions: actionsCopy,
		Targets: targetsCopy,
	}
}

// ToJSON converts a ViewRuleSet to a JSON byte slice.
func (v ViewDefinition) ToJSON() ([]byte, error) {
	return json.Marshal(v)
}

type ResponseHandlerParam struct {
	Method         string
	Path           string
	Handler        httpx.RequestHandler
	AllowedActions []Action
	Options        []HandlerOptions
}

type handlerOptions struct {
	skipViewDefValidation bool
}

type HandlerOptions func(*handlerOptions)

func SkipViewDefValidation(v bool) HandlerOptions {
	return func(o *handlerOptions) {
		o.skipViewDefValidation = v
	}
}
