package policy

import (
	"context"
	"fmt"
	"testing"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
)

// These tests cover a mixture of scenarios several of which are not even valid
// and will be caught by the validator. They are here to only validate the logic
func TestRules_IsActionAllowed(t *testing.T) {
	tests := []struct {
		name   string
		rules  Rules
		action Action
		target TargetResource
		want   bool
	}{
		{
			name:   "empty rules",
			rules:  Rules{},
			action: ActionResourceRead,
			target: "res://catalogs/test",
			want:   false,
		},
		{
			name: "global admin allows any action",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/test",
			want:   true,
		},
		{
			name: "catalog-specific admin allows actions",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/my-catalog/test",
			want:   true,
		},
		{
			name: "catalog-specific admin doesn't affect other catalogs",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/other-catalog/test",
			want:   false,
		},
		{
			name: "exact action and target match",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/test"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/test",
			want:   true,
		},
		{
			name: "wildcard target match",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/test",
			want:   true,
		},
		{
			name: "deny rule takes precedence",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/*"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/test"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/test",
			want:   false,
		},
		{
			name: "deny rule with wildcard",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/*"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/test/*"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/test/subpath",
			want:   false,
		},
		{
			name: "multiple actions in rule",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead, ActionResourceEdit},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			action: ActionResourceEdit,
			target: "res://catalogs/test",
			want:   true,
		},
		{
			name: "multiple targets in rule",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{
						"res://catalogs/test1",
						"res://catalogs/test2",
					},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/test2",
			want:   true,
		},
		{
			name: "no matching action",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceEdit},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/test",
			want:   false,
		},
		{
			name: "no matching target",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/other"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/test",
			want:   false,
		},
		{
			name: "deny intent without matching target",
			rules: Rules{
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/other"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/test",
			want:   false,
		},
		{
			name: "multiple rules with mixed intents",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/*"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/specific"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/specific",
			want:   false,
		},
		{
			name: "deny overrides catalog-specific admin",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/specific"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/my-catalog/specific",
			want:   false,
		},
		{
			name: "specific target with wildcard action",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead, ActionResourceEdit, ActionResourceDelete},
					Targets: []TargetResource{"res://catalogs/specific"},
				},
			},
			action: ActionResourceEdit,
			target: "res://catalogs/specific",
			want:   true,
		},
		{
			name: "multiple allow rules with deny",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/*"},
				},
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceEdit},
					Targets: []TargetResource{"res://catalogs/*"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead, ActionResourceEdit},
					Targets: []TargetResource{"res://catalogs/specific"},
				},
			},
			action: ActionResourceEdit,
			target: "res://catalogs/specific",
			want:   false,
		},
		{
			name: "deny overrides admin with multiple allow rules",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/*"},
				},
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/specific"},
				},
			},
			action: ActionResourceRead,
			target: "res://catalogs/my-catalog/specific",
			want:   false,
		},
		{
			name: "adopt view for catalog admin",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog"},
				},
			},
			action: ActionCatalogAdoptView,
			target: "res://catalogs/my-catalog/views/my-view",
			want:   true,
		},
		{
			name: "adopt view for catalog member",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdoptView},
					Targets: []TargetResource{"res://catalogs/my-catalog/*"},
				},
			},
			action: ActionCatalogAdoptView,
			target: "res://catalogs/my-catalog/views/my-view",
			want:   true,
		},
		{
			name: "adopt view for catalog member with deny",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdoptView},
					Targets: []TargetResource{"res://catalogs/my-catalog/*"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionCatalogAdoptView},
					Targets: []TargetResource{"res://catalogs/my-catalog/views/sensitive-view"},
				},
			},
			action: ActionCatalogAdoptView,
			target: "res://catalogs/my-catalog/views/sensitive-view",
			want:   false,
		},
		{
			name: "catalog admin overrides deny",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionCatalogAdoptView},
					Targets: []TargetResource{"res://catalogs/my-catalog/views/sensitive-view"},
				},
			},
			action: ActionCatalogAdoptView,
			target: "res://catalogs/my-catalog/views/sensitive-view",
			want:   false,
		},
		{
			name: "catalog admin overrides deny for list",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionCatalogAdoptView},
					Targets: []TargetResource{"res://catalogs/my-catalog/views/sensitive-view"},
				},
			},
			action: ActionCatalogList,
			target: "res://catalogs/my-catalog/views/sensitive-view",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "catalog admin overrides deny for list" {
				fmt.Println("catalog admin overrides deny for list")
			}
			if got, _ := tt.rules.IsActionAllowedOnResource(tt.action, tt.target); got != tt.want {
				t.Errorf("Rules.IsActionAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRules_IsSubsetOf(t *testing.T) {
	tests := []struct {
		name     string
		child    Rules
		parent   Rules
		expected bool
	}{
		{
			name:     "empty rule sets",
			child:    Rules{},
			parent:   Rules{},
			expected: true,
		},
		{
			name: "child is exact subset",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test"},
				},
			},
			expected: true,
		},
		{
			name: "child has more permissive target",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/*"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test"},
				},
			},
			expected: false,
		},
		{
			name: "child has less permissive target",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/*"},
				},
			},
			expected: true,
		},
		{
			name: "child has additional action",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead, ActionResourceEdit},
					Targets: []TargetResource{"res://catalogs/my-catalog/test"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test"},
				},
			},
			expected: false,
		},
		{
			name: "child has deny rule not in parent",
			child: Rules{
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/*"},
				},
			},
			expected: true,
		},
		{
			name: "child has admin rule not in parent",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/*"},
				},
			},
			expected: false,
		},
		{
			name: "child has multiple rules all within parent scope",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test1"},
				},
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test2"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/*"},
				},
			},
			expected: true,
		},
		{
			name: "child has mixed intents within parent scope",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test1"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test2"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/*"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test2"},
				},
			},
			expected: true,
		},
		{
			name: "child has variant rules within parent scope",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test/variants/v1"},
				},
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test/variants/v2"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/test/variants/*"},
				},
			},
			expected: true,
		},
		{
			name: "child has namespace rules within parent scope",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/namespaces/ns1/test"},
				},
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/namespaces/ns2/test"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/namespaces/*/test"},
				},
			},
			expected: false,
		},
		{
			name: "child has subset of namespace rules within parent scope",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/namespaces/ns1/test"},
				},
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/namespaces/ns2/test"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead, ActionResourceEdit},
					Targets: []TargetResource{"res://catalogs/my-catalog/namespaces/ns2/test"},
				},
			},
			expected: false,
		},
		{
			name: "child has view rules within parent scope",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/views/view1"},
				},
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/views/view2"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/views/*"},
				},
			},
			expected: true,
		},
		{
			name: "child has complex path rules within parent scope",
			child: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/namespaces/ns1/views/view1/variants/v1"},
				},
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/namespaces/ns1/views/view1/variants/v2"},
				},
			},
			parent: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/my-catalog/namespaces/ns1/views/view1/variants/*"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.child.IsSubsetOf(tt.parent)
			if got != tt.expected {
				t.Errorf("IsSubsetOf() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAreActionsAllowedOnResource(t *testing.T) {
	tests := []struct {
		name           string
		viewDefinition *ViewDefinition
		resource       string
		actions        []Action
		want           bool
		wantErr        bool
	}{
		{
			name: "empty actions",
			viewDefinition: &ViewDefinition{
				Scope: Scope{
					Catalog:   "test-catalog",
					Variant:   "test-variant",
					Namespace: "test-namespace",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{"res://resources/test-resource"},
					},
				},
			},
			resource: "/resources/test-resource",
			actions:  []Action{},
			want:     false,
			wantErr:  true,
		},
		{
			name: "all actions allowed",
			viewDefinition: &ViewDefinition{
				Scope: Scope{
					Catalog:   "test-catalog",
					Variant:   "test-variant",
					Namespace: "test-namespace",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead, ActionResourceEdit},
						Targets: []TargetResource{"res://resources/test-resource"},
					},
				},
			},
			resource: "/resources/test-resource",
			actions:  []Action{ActionResourceRead, ActionResourceEdit},
			want:     true,
			wantErr:  false,
		},
		{
			name: "some actions not allowed",
			viewDefinition: &ViewDefinition{
				Scope: Scope{
					Catalog:   "test-catalog",
					Variant:   "test-variant",
					Namespace: "test-namespace",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{"res://resources/test-resource"},
					},
				},
			},
			resource: "/resources/test-resource",
			actions:  []Action{ActionResourceRead, ActionResourceEdit},
			want:     false,
			wantErr:  false,
		},
		{
			name: "actions split across multiple rules",
			viewDefinition: &ViewDefinition{
				Scope: Scope{
					Catalog:   "test-catalog",
					Variant:   "test-variant",
					Namespace: "test-namespace",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{"res://resources/test-resource"},
					},
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{"res://resources/test-resource"},
					},
				},
			},
			resource: "/resources/test-resource",
			actions:  []Action{ActionResourceRead, ActionResourceEdit},
			want:     true,
			wantErr:  false,
		},
		{
			name: "deny rule takes precedence",
			viewDefinition: &ViewDefinition{
				Scope: Scope{
					Catalog:   "test-catalog",
					Variant:   "test-variant",
					Namespace: "test-namespace",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead, ActionResourceEdit},
						Targets: []TargetResource{"res://resources/test-resource"},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{"res://resources/test-resource"},
					},
				},
			},
			resource: "/resources/test-resource",
			actions:  []Action{ActionResourceRead, ActionResourceEdit},
			want:     false,
			wantErr:  false,
		},
		{
			name: "deny rule with wildcard target",
			viewDefinition: &ViewDefinition{
				Scope: Scope{
					Catalog:   "test-catalog",
					Variant:   "test-variant",
					Namespace: "test-namespace",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead, ActionResourceEdit},
						Targets: []TargetResource{"res://resources/*"},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{"res://resources/test-resource"},
					},
				},
			},
			resource: "/resources/test-resource",
			actions:  []Action{ActionResourceRead, ActionResourceEdit},
			want:     false,
			wantErr:  false,
		},
		{
			name:           "nil view definition",
			viewDefinition: nil,
			resource:       "/resources/test-resource",
			actions:        []Action{ActionResourceRead},
			want:           false,
			wantErr:        true,
		},
		{
			name: "empty rules",
			viewDefinition: &ViewDefinition{
				Scope: Scope{
					Catalog:   "test-catalog",
					Variant:   "test-variant",
					Namespace: "test-namespace",
				},
				Rules: Rules{},
			},
			resource: "/resources/test-resource",
			actions:  []Action{ActionResourceRead},
			want:     false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = catcommon.WithCatalogContext(ctx, &catcommon.CatalogContext{
				Catalog:   "test-catalog",
				Variant:   "test-variant",
				Namespace: "test-namespace",
			})
			_ = ctx
			// Debug output for canonicalized resource and rule target
			if tt.name == "some actions not allowed" {
				t.Logf("some actions not allowed")
			}

			got, _, err := AreActionsAllowedOnResource(tt.viewDefinition, tt.resource, tt.actions)
			if (err != nil) != tt.wantErr {
				t.Errorf("AreActionsAllowedOnResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AreActionsAllowedOnResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
