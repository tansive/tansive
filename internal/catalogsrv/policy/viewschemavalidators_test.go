package policy

import (
	"testing"

	"github.com/tansive/tansive/internal/catalogsrv/schema/schemavalidator"
)

func TestResourceURIValidator(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		// Valid cases - Catalogs
		{
			name:    "valid catalog resource",
			input:   "res://catalogs/my-catalog",
			isValid: true,
		},
		{
			name:    "valid catalog with nested path",
			input:   "res://catalogs/my-catalog/variants/my-variant",
			isValid: true,
		},
		{
			name:    "valid catalog with deeply nested path",
			input:   "res://catalogs/my-catalog/variants/my-variant/namespaces/my-namespace",
			isValid: true,
		},
		{
			name:    "valid catalog with collections",
			input:   "res://catalogs/my-catalog/variants/my-variant/collections/path",
			isValid: true,
		},
		{
			name:    "valid catalog with wildcard at end",
			input:   "res://catalogs/*",
			isValid: true,
		},
		{
			name:    "valid catalog with wildcard at end of nested path",
			input:   "res://catalogs/my-catalog/variants/*",
			isValid: true,
		},
		{
			name:    "valid catalog with trailing slash",
			input:   "res://catalogs/my-catalog/",
			isValid: true,
		},

		// Valid cases - Variants
		{
			name:    "valid variant resource",
			input:   "res://variants/my-variant",
			isValid: true,
		},
		{
			name:    "valid variant with nested path",
			input:   "res://variants/my-variant/namespaces/my-namespace",
			isValid: true,
		},
		{
			name:    "valid variant with wildcard at end",
			input:   "res://variants/*",
			isValid: true,
		},

		// Valid cases - Namespaces
		{
			name:    "valid namespace resource",
			input:   "res://namespaces/my-namespace",
			isValid: true,
		},
		{
			name:    "valid namespace with nested path",
			input:   "res://namespaces/my-namespace/collections/path",
			isValid: true,
		},
		{
			name:    "valid namespace with wildcard at end",
			input:   "res://namespaces/*",
			isValid: true,
		},

		// Valid cases - Resources
		{
			name:    "valid resource",
			input:   "res://resources/my-resource",
			isValid: true,
		},
		{
			name:    "valid resource with nested path",
			input:   "res://resources/my-resource/properties/definition",
			isValid: true,
		},
		{
			name:    "valid resource with wildcard at end",
			input:   "res://resources/*",
			isValid: true,
		},

		// Valid cases - Views
		{
			name:    "valid view resource",
			input:   "res://views/my-view",
			isValid: true,
		},
		{
			name:    "valid view with nested path",
			input:   "res://views/my-view/rules/allow",
			isValid: true,
		},
		{
			name:    "valid view with wildcard at end",
			input:   "res://views/*",
			isValid: true,
		},

		// Invalid cases - General
		{
			name:    "missing res:// prefix",
			input:   "catalogs/my-catalog",
			isValid: false,
		},
		{
			name:    "empty resource string",
			input:   "res://",
			isValid: true,
		},
		{
			name:    "invalid kind",
			input:   "res://invalid/my-resource",
			isValid: false,
		},
		{
			name:    "invalid path characters",
			input:   "res://catalogs/my@catalog",
			isValid: false,
		},
		{
			name:    "invalid path with spaces",
			input:   "res://catalogs/my catalog",
			isValid: false,
		},
		{
			name:    "invalid path with special characters",
			input:   "res://catalogs/my-catalog/variants/my@variant",
			isValid: false,
		},
		{
			name:    "invalid path with underscores",
			input:   "res://catalogs/my-catalog/variants/my_variant",
			isValid: false,
		},
		{
			name:    "wildcard in middle of path",
			input:   "res://catalogs/*/variants",
			isValid: false,
		},
		{
			name:    "wildcard followed by segments",
			input:   "res://catalogs/*/my-catalog",
			isValid: false,
		},
		{
			name:    "multiple wildcards",
			input:   "res://catalogs/*/variants/*",
			isValid: false,
		},
		{
			name:    "invalid segment after wildcard",
			input:   "res://catalogs/*/my catalog",
			isValid: false,
		},
		{
			name:    "empty segment in path",
			input:   "res://catalogs/my-catalog//variants",
			isValid: false,
		},
		{
			name:    "empty segment at start of path",
			input:   "res://catalogs//my-catalog",
			isValid: false,
		},
		{
			name:    "empty segment at end of path",
			input:   "res://catalogs/my-catalog//",
			isValid: false,
		},
		{
			name:    "invalid catalog with multiple slashes",
			input:   "res://catalogs/my-catalog//variants/my-variant",
			isValid: false,
		},
		{
			name:    "wildcard at start of path",
			input:   "res://*",
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResourceURI(tt.input)
			if (err == nil) != tt.isValid {
				t.Errorf("validateResourceURI() = %v, want %v for input %q", err == nil, tt.isValid, tt.input)
				if err != nil {
					t.Logf("Error message: %v", err)
				}
			}
		})
	}
}

func TestViewRuleIntentValidator(t *testing.T) {
	validate := schemavalidator.V()
	validate.RegisterValidation("viewRuleIntentValidator", validateViewRuleIntent)

	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{
			name:    "valid allow intent",
			input:   "Allow",
			isValid: true,
		},
		{
			name:    "valid deny intent",
			input:   "Deny",
			isValid: true,
		},
		{
			name:    "invalid intent - empty string",
			input:   "",
			isValid: false,
		},
		{
			name:    "invalid intent - random string",
			input:   "random",
			isValid: false,
		},
		{
			name:    "invalid intent - random string",
			input:   "allow",
			isValid: false,
		},
		{
			name:    "invalid intent - random string",
			input:   "deny",
			isValid: false,
		},
		{
			name:    "invalid intent - case sensitive",
			input:   "ALLOW",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Var(tt.input, "viewRuleIntentValidator")
			if (err == nil) != tt.isValid {
				t.Errorf("validateViewRuleIntent() = %v, want %v for input %q", err == nil, tt.isValid, tt.input)
			}
		})
	}
}

func TestViewRuleActionValidator(t *testing.T) {
	validate := schemavalidator.V()
	validate.RegisterValidation("viewRuleActionValidator", validateViewRuleAction)

	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{
			name:    "valid catalog admin action",
			input:   "system.catalog.admin",
			isValid: true,
		},
		{
			name:    "valid catalog list action",
			input:   "system.catalog.list",
			isValid: true,
		},
		{
			name:    "valid catalog adopt view action",
			input:   "system.catalog.adoptView",
			isValid: true,
		},
		{
			name:    "valid catalog create view action",
			input:   "system.catalog.createView",
			isValid: true,
		},
		{
			name:    "valid variant admin action",
			input:   "system.variant.admin",
			isValid: true,
		},
		{
			name:    "valid variant clone action",
			input:   "system.variant.clone",
			isValid: true,
		},
		{
			name:    "valid variant list action",
			input:   "system.variant.list",
			isValid: true,
		},
		{
			name:    "valid namespace create action",
			input:   "system.namespace.create",
			isValid: true,
		},
		{
			name:    "valid namespace list action",
			input:   "system.namespace.list",
			isValid: true,
		},
		{
			name:    "valid namespace admin action",
			input:   "system.namespace.admin",
			isValid: true,
		},
		{
			name:    "valid resource create action",
			input:   "system.resource.create",
			isValid: true,
		},
		{
			name:    "valid resource read action",
			input:   "system.resource.read",
			isValid: true,
		},
		{
			name:    "valid resource edit action",
			input:   "system.resource.edit",
			isValid: true,
		},
		{
			name:    "valid resource delete action",
			input:   "system.resource.delete",
			isValid: true,
		},
		{
			name:    "valid resource get action",
			input:   "system.resource.get",
			isValid: true,
		},
		{
			name:    "valid resource put action",
			input:   "system.resource.put",
			isValid: true,
		},
		{
			name:    "valid resource list action",
			input:   "system.resource.list",
			isValid: true,
		},
		{
			name:    "invalid action - empty string",
			input:   "",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Var(tt.input, "viewRuleActionValidator")
			if (err == nil) != tt.isValid {
				t.Errorf("validateViewRuleAction() = %v, want %v for input %q", err == nil, tt.isValid, tt.input)
			}
		})
	}
}
