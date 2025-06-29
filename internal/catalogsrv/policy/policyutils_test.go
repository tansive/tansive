package policy

import (
	"fmt"
	"testing"
)

func TestRules_MatchesAdmin(t *testing.T) {
	tests := []struct {
		name     string
		rules    Rules
		resource string
		want     bool
	}{
		{
			name:     "empty rules",
			rules:    Rules{},
			resource: "res://catalogs/test",
			want:     false,
		},
		{
			name: "matching catalog admin rule",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			resource: "res://catalogs/test",
			want:     true,
		},
		{
			name: "matching variant admin rule",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionVariantAdmin},
					Targets: []TargetResource{"res://variants/*"},
				},
			},
			resource: "res://variants/test",
			want:     true,
		},
		{
			name: "matching namespace admin rule",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionNamespaceAdmin},
					Targets: []TargetResource{"res://namespaces/*"},
				},
			},
			resource: "res://namespaces/test",
			want:     true,
		},
		{
			name: "non-matching resource type",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			resource: "res://variants/test",
			want:     false,
		},
		{
			name: "deny intent",
			rules: Rules{
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			resource: "res://catalogs/test",
			want:     false,
		},
		{
			name: "no admin actions",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			resource: "res://catalogs/test",
			want:     false,
		},
		{
			name: "multiple rules with one match",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://catalogs/*"},
				},
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			resource: "res://catalogs/test",
			want:     true,
		},
		{
			name: "exact resource match",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/test"},
				},
			},
			resource: "res://catalogs/test",
			want:     true,
		},
		{
			name: "wildcard match",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/*"},
				},
			},
			resource: "res://catalogs/test/subpath",
			want:     true,
		},
		{
			name: "long nested path with catalog admin",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/my-resource"},
				},
			},
			resource: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/my-resource",
			want:     false,
		},
		{
			name: "long nested path with variant admin",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionVariantAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/my-resource"},
				},
			},
			resource: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/my-resource",
			want:     false,
		},
		{
			name: "long nested path with namespace admin",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionNamespaceAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/my-resource"},
				},
			},
			resource: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/my-resource",
			want:     false,
		},
		{
			name: "exact path match without wildcards",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace"},
				},
			},
			resource: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace",
			want:     false,
		},
		{
			name: "mixed path lengths with catalog admin",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{
						"res://catalogs/my-catalog",
						"res://catalogs/my-catalog/variants/my-variant",
						"res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace",
					},
				},
			},
			resource: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace",
			want:     true,
		},
		{
			name: "multiple admin actions with different paths",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin, ActionVariantAdmin, ActionNamespaceAdmin},
					Targets: []TargetResource{
						"res://catalogs/my-catalog",
						"res://variants/my-variant",
						"res://namespaces/my-namespace",
					},
				},
			},
			resource: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace",
			want:     true,
		},
		{
			name: "path with numbers",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionNamespaceAdmin},
					Targets: []TargetResource{"res://catalogs/catalog-123/variants/variant-456/namespaces/namespace-789"},
				},
			},
			resource: "res://catalogs/catalog-123/variants/variant-456/namespaces/namespace-789",
			want:     true,
		},
		{
			name: "path with mixed case",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionNamespaceAdmin},
					Targets: []TargetResource{"res://catalogs/MyCatalog/variants/MyVariant/namespaces/MyNamespace"},
				},
			},
			resource: "res://catalogs/MyCatalog/variants/MyVariant/namespaces/MyNamespace/resources/my-resource",
			want:     true,
		},
		{
			name: "invalid resource kind - resources",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://resources/my-resource"},
				},
			},
			resource: "res://resources/my-resource",
			want:     false,
		},
		{
			name: "invalid resource kind - views",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://views/my-view"},
				},
			},
			resource: "res://views/my-view",
			want:     false,
		},
		{
			name: "invalid resource kind - unknown",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://unknown/type"},
				},
			},
			resource: "res://unknown/type",
			want:     false,
		},
		{
			name: "invalid resource kind in nested path",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog/resources/my-resource"},
				},
			},
			resource: "res://catalogs/my-catalog/resources/my-resource",
			want:     false,
		},
		{
			name: "mixed valid and invalid resource kinds",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{
						"res://catalogs/my-catalog",
						"res://resources/my-resource",
					},
				},
			},
			resource: "res://resources/my-resource",
			want:     false,
		},
		{
			name: "invalid resource kind with admin action",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionResourceRead},
					Targets: []TargetResource{"res://resources/my-resource"},
				},
			},
			resource: "res://resources/my-resource",
			want:     false,
		},
		{
			name: "invalid resource kind with multiple admin actions",
			rules: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin, ActionVariantAdmin, ActionNamespaceAdmin},
					Targets: []TargetResource{"res://resources/my-resource"},
				},
			},
			resource: "res://resources/my-resource",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "matching catalog admin rule" {
				fmt.Println("matching catalog admin rule")
			}
			if got, _ := tt.rules.matchesAdmin(tt.resource); got != tt.want {
				t.Errorf("Rules.matchesAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTargetResource_Matches(t *testing.T) {
	tests := []struct {
		name      string
		target    TargetResource
		actualRes string
		want      bool
	}{
		{
			name:      "exact match",
			target:    "res://catalogs/my-catalog",
			actualRes: "res://catalogs/my-catalog",
			want:      true,
		},
		{
			name:      "exact match with longer path",
			target:    "res://catalogs/my-catalog/variants/my-variant",
			actualRes: "res://catalogs/my-catalog/variants/my-variant",
			want:      true,
		},
		{
			name:      "wildcard at end matches longer path",
			target:    "res://catalogs/my-catalog/*",
			actualRes: "res://catalogs/my-catalog/variants/my-variant",
			want:      true,
		},
		{
			name:      "wildcard at end matches multiple segments",
			target:    "res://catalogs/my-catalog/*",
			actualRes: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace",
			want:      true,
		},
		{
			name:      "wildcard in middle matches single segment",
			target:    "res://catalogs/*/variants/my-variant",
			actualRes: "res://catalogs/my-catalog/variants/my-variant",
			want:      false,
		},
		{
			name:      "no match - different segments",
			target:    "res://catalogs/my-catalog",
			actualRes: "res://catalogs/other-catalog",
			want:      false,
		},
		{
			name:      "no match - shorter target without wildcard",
			target:    "res://catalogs/my-catalog",
			actualRes: "res://catalogs/my-catalog/variants/my-variant",
			want:      false,
		},
		{
			name:      "no match - longer target",
			target:    "res://catalogs/my-catalog/variants/my-variant",
			actualRes: "res://catalogs/my-catalog",
			want:      false,
		},
		{
			name:      "no match - different resource type",
			target:    "res://catalogs/my-catalog",
			actualRes: "res://variants/my-variant",
			want:      false,
		},
		{
			name:      "no match - invalid resource format",
			target:    "res://catalogs/my-catalog",
			actualRes: "invalid-resource",
			want:      false,
		},
		{
			name:      "no match - missing res:// prefix",
			target:    "res://catalogs/my-catalog",
			actualRes: "catalogs/my-catalog",
			want:      false,
		},
		{
			name:      "no match - wildcard at start",
			target:    "res://*/my-catalog",
			actualRes: "res://catalogs/my-catalog",
			want:      false,
		},
		{
			name:      "no match - wildcard in middle with longer path",
			target:    "res://catalogs/*/variants/my-variant",
			actualRes: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace",
			want:      false,
		},
		{
			name:      "no match - consecutive wildcards",
			target:    "res://catalogs/*/*/namespaces/*",
			actualRes: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace",
			want:      false,
		},
		{
			name:      "no match - wildcard with empty segment",
			target:    "res://catalogs/*/variants/my-variant",
			actualRes: "res://catalogs//variants/my-variant",
			want:      false,
		},
		{
			name:      "empty target",
			target:    "",
			actualRes: "res://catalogs/my-catalog",
			want:      false,
		},
		{
			name:      "empty actual",
			target:    "res://catalogs/my-catalog",
			actualRes: "",
			want:      false,
		},
		{
			name:      "both empty",
			target:    "",
			actualRes: "",
			want:      false,
		},
		{
			name:      "wildcard matches any single segment",
			target:    "res://catalogs/*/variants/my-variant",
			actualRes: "res://catalogs/any-catalog/variants/my-variant",
			want:      false,
		},
		{
			name:      "wildcard at end matches exactly one segment",
			target:    "res://catalogs/my-catalog/*",
			actualRes: "res://catalogs/my-catalog/variants",
			want:      true,
		},
		{
			name:      "no match - wildcard at end but path too long",
			target:    "res://catalogs/my-catalog/*",
			actualRes: "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.matches(tt.actualRes); got != tt.want {
				t.Errorf("TargetResource.matches() = %v, want %v for target %q and actual %q", got, tt.want, tt.target, tt.actualRes)
			}
		})
	}
}
