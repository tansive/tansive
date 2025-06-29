package policy

import (
	"fmt"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/schema/schemavalidator"
)

var validKinds map[string]struct{}

func init() {
	validKinds = make(map[string]struct{})
	for _, kind := range catcommon.ValidKindNames() {
		validKinds[kind] = struct{}{}
	}

	v := schemavalidator.V()
	v.RegisterValidation("viewRuleIntentValidator", validateViewRuleIntent)
	v.RegisterValidation("viewRuleActionValidator", validateViewRuleAction)
}

// validateViewRuleIntent checks if the effect is one of the allowed values.
func validateViewRuleIntent(fl validator.FieldLevel) bool {
	effect := Intent(fl.Field().String())
	return effect == IntentAllow || effect == IntentDeny
}

// validateViewRuleAction checks if the action is one of the allowed values.
func validateViewRuleAction(fl validator.FieldLevel) bool {
	action := Action(fl.Field().String())
	if action == "" {
		return false
	}
	if strings.HasPrefix(string(action), "system.") {
		return slices.Contains(ValidActions, action)
	}
	return true
}

// validateResourceURI validates that a resource URI follows the required structure.
// It expects a URI in the format "res://<kind>/<path>" where:
//   - <kind> must be one of the valid resource kinds (catalogs, variants, namespaces, etc.)
//   - <path> is an optional path that can contain multiple segments
//
// Returns an error if the URI is invalid, with a descriptive message about what went wrong.
//
// Example valid inputs:
//   - "res://catalogs/my-catalog"
//   - "res://variants/my-variant/namespaces/my-namespace"
//   - "res://resources/my-resource/properties/definition"
func validateResourceURI(uri string) error {
	const prefix = "res://"
	if len(uri) < len(prefix) || uri[:len(prefix)] != prefix {
		return fmt.Errorf("invalid resource URI: must start with %s", prefix)
	}
	rest := uri[len(prefix):]
	if rest == "" || rest == "*" || rest == "." {
		return nil
	}

	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 {
		return fmt.Errorf("invalid resource URI: missing resource kind")
	}

	kind := parts[0]
	if kind == "" {
		return fmt.Errorf("invalid resource URI: resource kind cannot be empty")
	}
	if _, ok := validKinds[kind]; !ok {
		return fmt.Errorf("invalid resource URI: unknown resource kind %q", kind)
	}

	if len(parts) == 2 {
		path := strings.TrimSuffix(parts[1], "/")
		segments := strings.Split(path, "/")

		// Validate each segment
		for i, segment := range segments {
			// Reject empty segments
			if segment == "" {
				return fmt.Errorf("invalid resource URI: empty path segment at position %d", i+1)
			}

			// Wildcard is only allowed as the last segment
			if segment == "*" {
				if i != len(segments)-1 {
					return fmt.Errorf("invalid resource URI: wildcard (*) is only allowed as the last path segment")
				}
				continue
			}

			// Validate non-wildcard segments
			if !schemavalidator.ValidatePathSegment(segment) {
				return fmt.Errorf("invalid resource URI: invalid path segment %q at position %d", segment, i+1)
			}
		}
	}

	return nil
}
