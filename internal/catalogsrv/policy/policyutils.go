package policy

import "strings"

// adminActionMap represents a set of admin actions
type adminActionMap map[Action]bool

// buildAdminActionMap creates a map of admin actions from a slice of actions
func buildAdminActionMap(actions []Action) adminActionMap {
	adminActions := make(adminActionMap)
	for _, action := range actions {
		switch action {
		case ActionCatalogAdmin, ActionVariantAdmin, ActionNamespaceAdmin:
			adminActions[action] = true
		}
	}
	return adminActions
}

func checkAdminMatch(resourceType string, ruleSegments []string) bool {
	lenRule := len(ruleSegments)
	if lenRule < 2 {
		return false
	}
	//namespaces/*
	if ruleSegments[lenRule-1] == "*" && ruleSegments[lenRule-2] == resourceType {
		return true
	}
	//namespaces/my-namespace/*
	if ruleSegments[lenRule-1] == "*" && lenRule > 3 {
		if ruleSegments[lenRule-3] == resourceType {
			return true
		}
	}
	//namespace/my-namespace
	return ruleSegments[lenRule-2] == resourceType
}

func (r Rules) matchesAdmin(resource string) (bool, Rule) {
	for _, rule := range r {
		if rule.Intent != IntentAllow {
			continue
		}

		adminActions := buildAdminActionMap(rule.Actions)
		if len(adminActions) == 0 {
			continue
		}

		for _, res := range rule.Targets {
			ruleSegments := strings.Split(string(res), "/")
			lenRule := len(ruleSegments)
			if lenRule < 2 {
				continue
			}
			isMatch := false
			if adminActions[ActionCatalogAdmin] && checkAdminMatch("catalogs", ruleSegments) {
				isMatch = true
			}
			if adminActions[ActionVariantAdmin] && checkAdminMatch("variants", ruleSegments) {
				isMatch = true
			}
			if adminActions[ActionNamespaceAdmin] && checkAdminMatch("namespaces", ruleSegments) {
				isMatch = true
			}
			if isMatch && (strings.HasPrefix(resource, string(res)) || res.matches(resource)) {
				return true, rule
			}
		}
	}
	return false, Rule{}
}

func (r TargetResource) matches(actualRes string) bool {
	if r == "" || actualRes == "" {
		return false
	}

	ruleSegments := strings.Split(string(r), "/")
	actualSegments := strings.Split(actualRes, "/")
	ruleLen := len(ruleSegments)
	actualLen := len(actualSegments)

	if ruleLen > actualLen {
		return false
	}

	if ruleLen < actualLen {
		if ruleSegments[ruleLen-1] != "*" {
			return false
		}
	}

	for i := range ruleLen {
		// this case should have been caught by the validator
		if ruleSegments[i] == "*" && i != ruleLen-1 {
			return false
		}
		if ruleSegments[i] == "*" || ruleSegments[i] == actualSegments[i] {
			continue
		}
		return false
	}

	return true
}
