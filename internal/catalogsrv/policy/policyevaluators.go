package policy

import (
	"context"
	"slices"
	"strings"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/httpx"
)

// IsActionAllowedOnResource evaluates whether a given action is permitted on a specific resource based on the rule set.
// It implements a deny-takes-precedence policy where deny rules override allow rules.
//
// Parameters:
//   - action: The Action to be evaluated
//   - target: The TargetResource to check permissions against
//
// Returns:
//   - bool: true if the action is allowed, false if denied
//   - map[Intent][]Rule: A map containing matched rules grouped by their intent (allow/deny)
//
// Note: This function first checks for admin matches, then evaluates regular rules.
// Deny rules take precedence over allow rules in case of conflicts.
func (ruleSet Rules) IsActionAllowedOnResource(action Action, target TargetResource) (bool, map[Intent][]Rule) {
	matchedRulesAllow := []Rule{}
	matchedRulesDeny := []Rule{}

	allowMatch := action == ActionAllow
	var matchedRule Rule
	// check if there is an admin match
	adminMatch, matchedRule := ruleSet.matchesAdmin(string(target))
	if adminMatch {
		allowMatch = true
		matchedRulesAllow = append(matchedRulesAllow, matchedRule)
	}
	// check if there is a match for the action
	for _, rule := range ruleSet {
		if slices.Contains(rule.Actions, action) {
			for _, res := range rule.Targets {
				switch rule.Intent {
				case IntentAllow:
					if res.matches(string(target)) {
						allowMatch = true
						matchedRulesAllow = append(matchedRulesAllow, rule)
					}
				case IntentDeny:
					if res.matches(string(target)) || // target is allowed by the rule
						target.matches(string(res)) { // target is more permissive than the rule when we evaluate rule subsets
						allowMatch = false
						matchedRulesDeny = append(matchedRulesDeny, rule)
					}
				}
			}
		}
	}
	return allowMatch, map[Intent][]Rule{
		IntentAllow: matchedRulesAllow,
		IntentDeny:  matchedRulesDeny,
	}
}

// IsSubsetOf determines if this RuleSet is a proper subset of another RuleSet.
// A RuleSet is considered a subset if all its actions and targets are permissible
// under the other set's rules.
//
// Parameters:
//   - other: The RuleSet to compare against
//
// Returns:
//   - bool: true if this RuleSet is a subset of the other set, false otherwise
//
// Note: This function only considers allow rules in the comparison.
// All actions and targets in this set must be explicitly allowed by the other set.
func (ruleSet Rules) IsSubsetOf(other Rules) bool {
	for _, rule := range ruleSet {
		for _, action := range rule.Actions {
			for _, target := range rule.Targets {
				if rule.Intent == IntentAllow {
					allow, _ := other.IsActionAllowedOnResource(action, target)
					if !allow {
						return false
					}
				}
			}
		}
	}
	return true
}

// ValidateDerivedView ensures that a derived view is valid with respect to its parent view.
// It performs two key validations:
// 1. Ensures the derived view's scope matches the parent's scope
// 2. Verifies that all rules in the derived view are permissible by the parent view
//
// Parameters:
//   - ctx: The context for the operation
//   - parent: The parent ViewDefinition to validate against
//   - child: The derived ViewDefinition to validate
//
// Returns:
//   - apperrors.Error: nil if validation succeeds, otherwise returns an appropriate error
//
// Note: Both parent and child views are canonicalized before validation.
// Returns ErrInvalidView if either view is nil or if the derived view's rules
// are not a subset of the parent view's rules.
func ValidateDerivedView(ctx context.Context, parent *ViewDefinition, child *ViewDefinition) apperrors.Error {
	if parent == nil || child == nil {
		return ErrInvalidView
	}

	parent = canonicalizeViewDefinition(parent)
	child = canonicalizeViewDefinition(child)

	if !child.Rules.IsSubsetOf(parent.Rules) {
		return ErrInvalidView.New("derived view rules must be a subset of parent view rules")
	}

	return nil
}

// AreActionsAllowedOnResource checks if a set of actions are permitted on a specific resource
// according to the given view definition.
//
// Parameters:
//   - vd: The ViewDefinition containing the rules to evaluate
//   - resource: The resource path to check permissions for
//   - actions: The list of actions to validate
//
// Returns:
//   - bool: true if all actions are allowed, false if any action is denied
//   - apperrors.Error: nil if the check succeeds, otherwise returns an appropriate error
//
// Note: The function validates that the view definition, resource, and actions are non-empty.
// It resolves the target scope and resource before performing the permission check.
// All actions must be allowed for the function to return true.
func AreActionsAllowedOnResource(vd *ViewDefinition, resource string, actions []Action) (bool, map[Intent][]Rule, apperrors.Error) {
	if vd == nil {
		return false, nil, ErrInvalidView.Msg("view definition is nil")
	}
	if resource == "" {
		return false, nil, ErrInvalidView.Msg("resource is empty")
	}
	if len(actions) == 0 {
		return false, nil, ErrInvalidView.Msg("actions are empty")
	}

	targetResource, err := resolveTargetResource(vd.Scope, resource)
	if err != nil {
		return false, nil, ErrInvalidView.New(err.Error())
	}

	vd = canonicalizeViewDefinition(vd)
	var basis map[Intent][]Rule

	for _, action := range actions {
		allowed := false
		allowed, basis = vd.Rules.IsActionAllowedOnResource(action, targetResource)
		if !allowed {
			return false, basis, nil
		}
	}
	return true, basis, nil
}

// CanAdoptView determines if the current view has permission to adopt another view
// within the catalog context.
//
// Parameters:
//   - ctx: The context for the operation
//   - view: The name of the view to check adoption permissions for
//
// Returns:
//   - bool: true if the current view can adopt the specified view, false otherwise
//   - apperrors.Error: nil if the check succeeds, otherwise returns an appropriate error
//
// Note: This function requires a valid catalog context and an authorized view definition.
// It checks if the current view has the ActionCatalogAdoptView permission for the target view.
func CanAdoptView(ctx context.Context, view string) (bool, apperrors.Error) {
	catalog := catcommon.GetCatalog(ctx)
	if catalog == "" {
		return false, ErrInvalidView.Msg("unable to resolve catalog")
	}
	viewResource, _ := resolveTargetResource(Scope{Catalog: catalog}, "/views/"+view)
	ourViewDef, err := ResolveAuthorizedViewDef(ctx)
	if err != nil {
		return false, ErrInvalidView.Msg(err.Error())
	}
	if ourViewDef == nil {
		return false, ErrInvalidView.Msg("unable to resolve view definition")
	}
	allowed, _ := ourViewDef.Rules.IsActionAllowedOnResource(ActionCatalogAdoptView, viewResource)
	return allowed, nil
}

// CanUseSkillSet checks if the current view has permission to use a skill set
// within the catalog context.
//
// Parameters:
//   - ctx: The context for the operation
//   - skillSetPath: The path of the skill set to check usage permissions for
//
// Returns:
//   - bool: true if the current view can use the specified skill set, false otherwise
//   - apperrors.Error: nil if the check succeeds, otherwise returns an appropriate error
//
// Note: This function requires a valid catalog context and an authorized view definition.
// It checks if the current view has the ActionSkillSetUse permission for the target skill set.
func CanUseSkillSet(ctx context.Context, skillSetPath string) (bool, apperrors.Error) {
	vd := GetViewDefinition(ctx)
	if vd == nil {
		return false, ErrInvalidView.Msg("unable to resolve view definition")
	}
	skillSetResource, _ := resolveTargetResource(vd.Scope, "/skillsets/"+skillSetPath)
	ourViewDef, err := ResolveAuthorizedViewDef(ctx)
	if err != nil {
		return false, ErrInvalidView.Msg(err.Error())
	}
	if ourViewDef == nil {
		return false, ErrInvalidView.Msg("unable to resolve view definition")
	}
	allowed, _ := ourViewDef.Rules.IsActionAllowedOnResource(ActionSkillSetUse, skillSetResource)
	return allowed, nil
}

// CanAdoptViewAsUser checks if the current user has permission to adopt a view
// within the catalog context. We current allow by default in single user mode.
//
// Parameters:
//   - ctx: The context for the operation
//   - view: The name of the view to check adoption permissions for
func CanAdoptViewAsUser(ctx context.Context, view string) bool {
	if catcommon.GetSubjectType(ctx) != catcommon.SubjectTypeUser {
		return false
	}
	if config.Config().SingleUserMode {
		return true
	}
	return false
}

func getResourceKindFromPath(resourcePath string) string {
	path := strings.Trim(resourcePath, "/")
	segments := strings.Split(path, "/")
	var resourceKind string
	if len(segments) > 0 {
		resourceKind = segments[0]
	}
	return resourceKind
}

func normalizeResourcePath(resourceKind string, resource TargetResource) TargetResource {
	if resourceKind == catcommon.KindNameResources {
		const prefix = "/resources/definition"
		if strings.HasPrefix(string(resource), prefix) {
			// Rewrite /resources/definition/... → /resources/...
			return TargetResource("/resources" + strings.TrimPrefix(string(resource), prefix))
		}
	}
	return resource
}

func resolveTargetResource(scope Scope, resourcePath string) (TargetResource, error) {
	resourcePath = strings.TrimPrefix(resourcePath, "res://")
	targetResource := TargetResource(resourcePath)
	targetResource = normalizeResourcePath(getResourceKindFromPath(resourcePath), targetResource)
	targetResource = canonicalizeResourcePath(scope, TargetResource("res://"+strings.TrimPrefix(string(targetResource), "/")))
	if targetResource == "" {
		return "", httpx.ErrApplicationError("unable to canonicalize resource path")
	}
	return targetResource, nil
}

func ResolveAuthorizedViewDef(ctx context.Context) (*ViewDefinition, error) {
	c := catcommon.GetCatalogContext(ctx)
	if c == nil {
		return nil, httpx.ErrUnAuthorized("missing request context")
	}
	// Get the authorized view definition from the context
	authorizedViewDef := canonicalizeViewDefinition(GetViewDefinition(ctx))
	if authorizedViewDef == nil {
		return nil, httpx.ErrUnAuthorized("unable to resolve view definition")
	}
	return authorizedViewDef, nil
}

func resolveTargetScope(ctx context.Context) (Scope, error) {
	c := catcommon.GetCatalogContext(ctx)
	if c == nil {
		return Scope{}, httpx.ErrUnAuthorized("missing request context")
	}
	return Scope{
		Catalog:   c.Catalog,
		Variant:   c.Variant,
		Namespace: c.Namespace,
	}, nil
}
