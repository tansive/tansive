package policy

import (
	"path"
	"strings"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
)

// removeDuplicates removes duplicate elements from a slice while preserving order.
// Returns a new slice containing only unique elements in their original order.
func removeDuplicates[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return slice
	}

	seen := make(map[T]struct{}, len(slice))
	unique := make([]T, 0, len(slice))

	for _, v := range slice {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			unique = append(unique, v)
		}
	}

	// If no duplicates were found, return the original slice
	if len(unique) == len(slice) {
		return slice
	}

	return unique
}

// deduplicateRules removes duplicate actions and targets from each rule in the ViewRuleSet.
// Returns a new ViewRuleSet with all duplicates removed while preserving the original order.
func deduplicateRules(rules Rules) Rules {
	if len(rules) == 0 {
		return rules
	}

	result := make(Rules, len(rules))
	for i, rule := range rules {
		result[i] = Rule{
			Intent:  rule.Intent,
			Actions: removeDuplicates(rule.Actions),
			Targets: removeDuplicates(rule.Targets),
		}
	}
	return result
}

// canonicalizeResourcePath transforms a resource string based on the provided scope.
// It handles the conversion of resource paths and ensures proper formatting
// of catalog, variant, workspace, and namespace components.
func canonicalizeResourcePath(scope Scope, resource TargetResource) TargetResource {
	s := string(resource)
	s = strings.TrimPrefix(s, "res://")
	s = strings.TrimPrefix(s, "/") // just in case, the res:// prefix was missing
	catalogLevel := catcommon.IsCatalogLevelKind(getResourceKindFromPath(s))
	metadataPath := strings.Builder{}
	if scope.Catalog != "" {
		metadataPath.WriteString("catalogs/")
		metadataPath.WriteString(scope.Catalog + "/")
	}
	if scope.Variant != "" && !catalogLevel {
		metadataPath.WriteString("variants/")
		metadataPath.WriteString(scope.Variant + "/")
	}
	if scope.Namespace != "" && !catalogLevel {
		metadataPath.WriteString("namespaces/")
		metadataPath.WriteString(scope.Namespace + "/")
	}
	canonicalized := path.Clean(metadataPath.String() + strings.TrimSuffix(s, "/"))
	if canonicalized == "." || canonicalized == "/" {
		canonicalized = ""
	}
	canonicalized = "res://" + canonicalized

	return TargetResource(canonicalized)
}

// canonicalizeViewDefinition canonicalizes all targets in the view definition to its scope
func canonicalizeViewDefinition(v *ViewDefinition) *ViewDefinition {
	if v == nil {
		return nil
	}
	vd := v.DeepCopy()
	for i, rule := range vd.Rules {
		if len(rule.Targets) == 0 {
			rule.Targets = []TargetResource{""}
			vd.Rules[i] = rule
		}
		for j, target := range rule.Targets {
			vd.Rules[i].Targets[j] = TargetResource(canonicalizeResourcePath(vd.Scope, target))
		}
	}
	return &vd
}
