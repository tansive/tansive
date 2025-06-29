package policy

import (
	"net/http"
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/httpx"
)

// EnforceViewPolicyMiddleware creates a middleware that enforces view-based access control policies.
// It evaluates whether the current request is allowed based on the authorized view definition,
// target scope, and requested resource path.
//
// The middleware performs the following steps:
// 1. Resolves the authorized view definition from the request context
// 2. Determines the target scope (catalog, variant, namespace)
// 3. Resolves the target resource from the request URL path
// 4. Evaluates each allowed action against the policy rules
// 5. Logs the policy decision with detailed information
//
// Parameters:
//   - handler: ResponseHandlerParam containing the allowed actions and the actual request handler
//
// Returns:
//   - httpx.RequestHandler: A middleware function that enforces the policy
//
// Note: The middleware implements a first-match policy where access is granted if any
// of the allowed actions are permitted. It logs detailed policy decisions including
// matched allow and deny rules for auditing purposes.
// Returns ErrDisallowedByPolicy if no allowed actions are permitted by the policy.
func EnforceViewPolicyMiddleware(handler ResponseHandlerParam) httpx.RequestHandler {
	return func(r *http.Request) (*httpx.Response, error) {
		ctx := r.Context()

		options := &handlerOptions{}
		for _, opt := range handler.Options {
			opt(options)
		}

		if options.skipViewDefValidation {
			if handler.AllowedActions != nil && slices.Contains(handler.AllowedActions, ActionAllow) {
				log.Ctx(ctx).Info().Msg("Skipping view definition validation for allow action")
				return handler.Handler(r)
			}
		}

		// Resolve the authorized view definition
		authorizedViewDef, err := ResolveAuthorizedViewDef(ctx)
		if err != nil {
			return nil, err
		}

		// Resolve the target scope
		targetScope, err := resolveTargetScope(ctx)
		if err != nil {
			return nil, err
		}

		// Resolve the target resource
		targetResource, err := resolveTargetResource(targetScope, r.URL.Path)
		if err != nil {
			return nil, err
		}

		// Validate against the policy
		allowed := false
		matchedRules := map[Intent][]Rule{
			IntentAllow: {},
			IntentDeny:  {},
		}
		for _, action := range handler.AllowedActions {
			isAllowed, ruleSet := authorizedViewDef.Rules.IsActionAllowedOnResource(action, targetResource)

			// Track rules
			for intent, rules := range ruleSet {
				matchedRules[intent] = append(matchedRules[intent], rules...)
			}

			if isAllowed {
				allowed = true
				break
			}
		}

		// log the policy decision
		logger := log.Ctx(ctx).With().
			Str("event_type", "policy_decision").
			Str("target_resource", string(targetResource)).
			Interface("handler_actions", handler.AllowedActions).
			Bool("allowed", allowed).
			Interface("matched_allow_rules", matchedRules[IntentAllow]).
			Interface("matched_deny_rules", matchedRules[IntentDeny]).
			Logger()

		if !allowed {
			logger.Warn().Msg("access denied")
			return nil, ErrDisallowedByPolicy
		}
		logger.Info().Msg("access allowed")

		// If we get here, we are good to go, so call the handler
		return handler.Handler(r)
	}
}
