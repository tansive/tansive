package policy

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
		{
			name:     "duplicates at start and end",
			input:    []string{"a", "b", "c", "a"},
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicates(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeduplicateRules(t *testing.T) {
	tests := []struct {
		name     string
		input    Rules
		expected Rules
	}{
		{
			name:     "empty rules",
			input:    Rules{},
			expected: Rules{},
		},
		{
			name: "no duplicates",
			input: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin, ActionCatalogList},
					Targets: []TargetResource{"res://catalogs/my-catalog", "res://catalogs/other-catalog"},
				},
			},
			expected: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin, ActionCatalogList},
					Targets: []TargetResource{"res://catalogs/my-catalog", "res://catalogs/other-catalog"},
				},
			},
		},
		{
			name: "with duplicate actions and targets",
			input: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin, ActionCatalogList, ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog", "res://catalogs/my-catalog", "res://catalogs/other-catalog"},
				},
			},
			expected: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin, ActionCatalogList},
					Targets: []TargetResource{"res://catalogs/my-catalog", "res://catalogs/other-catalog"},
				},
			},
		},
		{
			name: "multiple rules with duplicates",
			input: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin, ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog", "res://catalogs/my-catalog"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionCatalogList, ActionCatalogList},
					Targets: []TargetResource{"res://catalogs/other-catalog", "res://catalogs/other-catalog"},
				},
			},
			expected: Rules{
				{
					Intent:  IntentAllow,
					Actions: []Action{ActionCatalogAdmin},
					Targets: []TargetResource{"res://catalogs/my-catalog"},
				},
				{
					Intent:  IntentDeny,
					Actions: []Action{ActionCatalogList},
					Targets: []TargetResource{"res://catalogs/other-catalog"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateRules(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanonicalizeResourcePath(t *testing.T) {
	// Note: Resource paths should be defined relative to the scope.
	// For example, if the scope is catalog/my-catalog/variant/my-variant,
	// then resource paths should be like "resources/my-resource" rather than
	// "catalogs/other-catalog/resources/my-resource". The latter will result in
	// an invalid resource pointer and is considered incorrect usage.

	tests := []struct {
		name     string
		scope    Scope
		resource TargetResource
		want     TargetResource
	}{
		{
			name: "full scope with simple resource",
			scope: Scope{
				Catalog:   "my-catalog",
				Variant:   "my-variant",
				Namespace: "my-namespace",
			},
			resource: "res://resources/my-resource",
			want:     "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/my-resource",
		},
		{
			name: "catalog only scope",
			scope: Scope{
				Catalog: "my-catalog",
			},
			resource: "res://resources/my-resource",
			want:     "res://catalogs/my-catalog/resources/my-resource",
		},
		{
			name: "catalog and variant scope",
			scope: Scope{
				Catalog: "my-catalog",
				Variant: "my-variant",
			},
			resource: "res://resources/my-resource",
			want:     "res://catalogs/my-catalog/variants/my-variant/resources/my-resource",
		},
		{
			name: "catalog and namespace scope",
			scope: Scope{
				Catalog:   "my-catalog",
				Namespace: "my-namespace",
			},
			resource: "res://resources/my-resource",
			want:     "res://catalogs/my-catalog/namespaces/my-namespace/resources/my-resource",
		},
		{
			name:     "empty scope",
			scope:    Scope{},
			resource: "res://resources/my-resource",
			want:     "res://resources/my-resource",
		},
		{
			name: "resource with trailing slash",
			scope: Scope{
				Catalog: "my-catalog",
			},
			resource: "res://resources/my-resource/",
			want:     "res://catalogs/my-catalog/resources/my-resource",
		},
		{
			name: "resource with nested path",
			scope: Scope{
				Catalog: "my-catalog",
				Variant: "my-variant",
			},
			resource: "res://resources/my-resource/properties/definition",
			want:     "res://catalogs/my-catalog/variants/my-variant/resources/my-resource/properties/definition",
		},
		{
			name: "resource with wildcard",
			scope: Scope{
				Catalog: "my-catalog",
			},
			resource: "res://resources/*",
			want:     "res://catalogs/my-catalog/resources/*",
		},
		{
			name: "resource with multiple segments",
			scope: Scope{
				Catalog:   "my-catalog",
				Variant:   "my-variant",
				Namespace: "my-namespace",
			},
			resource: "res://resources/my-resource/properties/definition/schema",
			want:     "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/my-resource/properties/definition/schema",
		},
		{
			name: "resource without res:// prefix",
			scope: Scope{
				Catalog: "my-catalog",
			},
			resource: "resources/my-resource",
			want:     "res://catalogs/my-catalog/resources/my-resource",
		},
		{
			name: "resource with empty segments",
			scope: Scope{
				Catalog: "my-catalog",
			},
			resource: "res://resources//my-resource",
			want:     "res://catalogs/my-catalog/resources/my-resource",
		},
		{
			name: "empty resource string",
			scope: Scope{
				Catalog: "my-catalog",
			},
			resource: "",
			want:     "res://catalogs/my-catalog",
		},
		{
			name: "incorrect usage - resource with catalog in path",
			scope: Scope{
				Catalog: "my-catalog",
				Variant: "my-variant",
			},
			resource: "res://catalogs/other-catalog/resources/my-resource",
			want:     "res://catalogs/my-catalog/variants/my-variant/catalogs/other-catalog/resources/my-resource",
		},
		{
			name: "resource with variant wildcard",
			scope: Scope{
				Catalog: "my-catalog",
				Variant: "my-variant",
			},
			resource: "res://*",
			want:     "res://catalogs/my-catalog/variants/my-variant/*",
		},
		{
			name: "resource with variant wildcard",
			scope: Scope{
				Catalog: "my-catalog",
				Variant: "my-variant",
			},
			resource: "res://.",
			want:     "res://catalogs/my-catalog/variants/my-variant",
		},
		{
			name: "resource with catalog wildcard",
			scope: Scope{
				Catalog: "my-catalog",
			},
			resource: "res://*",
			want:     "res://catalogs/my-catalog/*",
		},
		{
			name: "resource with catalog wildcard",
			scope: Scope{
				Catalog: "my-catalog",
			},
			resource: "res://.",
			want:     "res://catalogs/my-catalog",
		},
		{
			name: "resource with resource wildcard",
			scope: Scope{
				Catalog: "my-catalog",
				Variant: "my-variant",
			},
			resource: "res://resources/my-resource/*",
			want:     "res://catalogs/my-catalog/variants/my-variant/resources/my-resource/*",
		},
		// Note: The following tests are incorrect usage and will be rejected by the validator.
		{
			name: "resource starting with slash",
			scope: Scope{
				Catalog: "my-catalog",
			},
			resource: "/resources/my-resource",
			want:     "res://catalogs/my-catalog/resources/my-resource",
		},
		{
			name: "resource with multiple consecutive slashes",
			scope: Scope{
				Catalog: "my-catalog",
				Variant: "my-variant",
			},
			resource: "res://resources//my-resource///properties//definition",
			want:     "res://catalogs/my-catalog/variants/my-variant/resources/my-resource/properties/definition",
		},
		{
			name: "view resource",
			scope: Scope{
				Catalog:   "my-catalog",
				Variant:   "my-variant",
				Namespace: "my-namespace",
			},
			resource: "/views/my-view",
			want:     "res://catalogs/my-catalog/views/my-view",
		},
		{
			name:     "empty scope with empty resource",
			scope:    Scope{},
			resource: "",
			want:     "res://",
		},
		{
			name:     "empty scope with just res://",
			scope:    Scope{},
			resource: "res://",
			want:     "res://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalizeResourcePath(tt.scope, tt.resource)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCanonicalizeViewDefinition(t *testing.T) {
	tests := []struct {
		name     string
		input    *ViewDefinition
		expected *ViewDefinition
	}{
		{
			name: "multiple rules with different targets",
			input: &ViewDefinition{
				Scope: Scope{
					Catalog:   "my-catalog",
					Variant:   "my-variant",
					Namespace: "my-namespace",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://resources/*",
							"res://resources/specific/*",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://resources/sensitive/*",
							"res://resources/confidential/*",
						},
					},
				},
			},
			expected: &ViewDefinition{
				Scope: Scope{
					Catalog:   "my-catalog",
					Variant:   "my-variant",
					Namespace: "my-namespace",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/*",
							"res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/specific/*",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/sensitive/*",
							"res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace/resources/confidential/*",
						},
					},
				},
			},
		},
		{
			name: "multiple rules with catalog-only scope",
			input: &ViewDefinition{
				Scope: Scope{
					Catalog: "my-catalog",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://resources/*",
							"res://resources/specific/*",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://resources/sensitive/*",
						},
					},
				},
			},
			expected: &ViewDefinition{
				Scope: Scope{
					Catalog: "my-catalog",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://catalogs/my-catalog/resources/*",
							"res://catalogs/my-catalog/resources/specific/*",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://catalogs/my-catalog/resources/sensitive/*",
						},
					},
				},
			},
		},
		{
			name: "multiple rules with catalog and variant scope",
			input: &ViewDefinition{
				Scope: Scope{
					Catalog: "my-catalog",
					Variant: "my-variant",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://resources/*",
							"res://resources/specific/*",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://resources/sensitive/*",
						},
					},
				},
			},
			expected: &ViewDefinition{
				Scope: Scope{
					Catalog: "my-catalog",
					Variant: "my-variant",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://catalogs/my-catalog/variants/my-variant/resources/*",
							"res://catalogs/my-catalog/variants/my-variant/resources/specific/*",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://catalogs/my-catalog/variants/my-variant/resources/sensitive/*",
						},
					},
				},
			},
		},
		{
			name: "empty scope",
			input: &ViewDefinition{
				Scope: Scope{},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://resources/*",
							"res://resources/specific/*",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://resources/sensitive/*",
						},
					},
				},
			},
			expected: &ViewDefinition{
				Scope: Scope{},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://resources/*",
							"res://resources/specific/*",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://resources/sensitive/*",
						},
					},
				},
			},
		},
		{
			name: "empty resources in rules",
			input: &ViewDefinition{
				Scope: Scope{
					Catalog: "my-catalog",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"",
							"res://",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://",
						},
					},
				},
			},
			expected: &ViewDefinition{
				Scope: Scope{
					Catalog: "my-catalog",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://catalogs/my-catalog",
							"res://catalogs/my-catalog",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://catalogs/my-catalog",
						},
					},
				},
			},
		},
		{
			name: "empty target",
			input: &ViewDefinition{
				Scope: Scope{
					Catalog: "my-catalog",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://",
						},
					},
				},
			},
			expected: &ViewDefinition{
				Scope: Scope{
					Catalog: "my-catalog",
				},
				Rules: Rules{
					{
						Intent:  IntentAllow,
						Actions: []Action{ActionResourceRead},
						Targets: []TargetResource{
							"res://catalogs/my-catalog",
						},
					},
					{
						Intent:  IntentDeny,
						Actions: []Action{ActionResourceEdit},
						Targets: []TargetResource{
							"res://catalogs/my-catalog",
						},
					},
				},
			},
		},
		{
			name:     "nil view definition",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "empty target" {
				fmt.Println("empty target")
			}
			got := canonicalizeViewDefinition(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
