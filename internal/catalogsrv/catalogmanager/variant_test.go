package catalogmanager

import (
	"testing"

	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestNewVariantManager(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected string
	}{
		{
			name: "valid variant",
			jsonData: `
{
    "apiVersion": "0.1.0-alpha.1",
    "kind": "Variant",
    "metadata": {
        "name": "valid-variant",
        "catalog": "validcatalog",
        "description": "This is a valid variant"
    }
}`,
			expected: "",
		},
		{
			name: "invalid version",
			jsonData: `
{
    "apiVersion": "v2",
    "kind": "Variant",
    "metadata": {
        "name": "invalid-version-variant",
        "catalog": "validcatalog",
        "description": "Invalid version in variant"
    }
}`,
			expected: ErrInvalidVersion.Error(),
		},
		{
			name: "invalid kind",
			jsonData: `
{
    "apiVersion": "0.1.0-alpha.1",
    "kind": "InvalidKind",
    "metadata": {
        "name": "invalid-kind-variant",
        "catalog": "validcatalog",
        "description": "Invalid kind in variant"
    }
}`,
			expected: ErrInvalidKind.Error(),
		},
		{
			name:     "empty JSON data",
			jsonData: "",
			expected: ErrInvalidSchema.Error(),
		},
	}

	// Initialize context with logger and database connection
	ctx := newDb()
	defer db.DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("PDEFGH")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant and project for testing
	err := db.DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer db.DB(ctx).DeleteTenant(ctx, tenantID)

	err = db.DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog for testing the variants
	catalogName := "validcatalog"
	err = db.DB(ctx).CreateCatalog(ctx, &models.Catalog{
		Name:        catalogName,
		Description: "Test catalog",
		ProjectID:   projectID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	assert.NoError(t, err)
	defer db.DB(ctx).DeleteCatalog(ctx, uuid.Nil, catalogName)

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			// Convert JSON to []byte
			jsonData := []byte(tt.jsonData)

			// Create a new variant manager
			vm, err := NewVariantManager(ctx, jsonData, "", catalogName)
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}

			// Check if the error string matches the expected error string
			if errStr != tt.expected {
				t.Errorf("got error %v, expected error %v", err, tt.expected)
			} else if tt.expected == "" {
				// If no error is expected, validate variant properties
				assert.NotNil(t, vm)
				assert.Equal(t, "valid-variant", vm.Name())
				assert.Equal(t, "This is a valid variant", vm.Description())

				// Save the variant
				err = vm.Save(ctx)
				assert.NoError(t, err)

				// Attempt to save again to check for duplicate handling
				err = vm.Save(ctx)
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrAlreadyExists)

				// Load the variant
				loadedVariant, loadErr := LoadVariantManager(ctx, vm.CatalogID(), uuid.Nil, vm.Name())
				assert.NoError(t, loadErr)
				assert.Equal(t, vm.Name(), loadedVariant.Name())
				assert.Equal(t, vm.Description(), loadedVariant.Description())

				// Load the variant with ID
				loadedVariant, loadErr = LoadVariantManager(ctx, uuid.Nil, vm.ID(), vm.Name())
				assert.NoError(t, loadErr)
				assert.Equal(t, vm.Name(), loadedVariant.Name())
				assert.Equal(t, vm.Description(), loadedVariant.Description())

				// Load the variant with an invalid name
				_, loadErr = LoadVariantManager(ctx, vm.CatalogID(), uuid.Nil, "InvalidVariant")
				assert.Error(t, loadErr)
				assert.ErrorIs(t, loadErr, ErrVariantNotFound)

				// Delete the variant
				err = DeleteVariant(ctx, vm.CatalogID(), vm.ID(), vm.Name())
				assert.NoError(t, err)

				// Try loading the deleted variant
				_, loadErr = LoadVariantManager(ctx, vm.CatalogID(), vm.ID(), vm.Name())
				assert.Error(t, loadErr)
				assert.ErrorIs(t, loadErr, ErrVariantNotFound)

				// Try deleting again to ensure no error is raised
				err = DeleteVariant(ctx, vm.CatalogID(), vm.ID(), vm.Name())
				assert.NoError(t, err)
			}
		})
	}
}
