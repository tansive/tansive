package catalogmanager

import (
	"encoding/json"
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
)

func TestNewCatalogManager(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected error
	}{
		{
			name: "valid catalog",
			jsonData: `
{
    "apiVersion": "0.1.0-alpha.1",
    "kind": "Catalog",
    "metadata": {
        "name": "valid-catalog",
        "description": "This is a valid catalog"
    }
}`,
			expected: nil,
		},
		{
			name: "invalid version",
			jsonData: `
{
    "apiVersion": "v2",
    "kind": "Catalog",
    "metadata": {
        "name": "invalid-version-catalog",
        "description": "Invalid version in catalog"
    }
}`,
			expected: ErrInvalidSchema,
		},
		{
			name: "invalid kind",
			jsonData: `
{
    "apiVersion": "0.1.0-alpha.1",
    "kind": "InvalidKind",
    "metadata": {
        "name": "invalid-kind-catalog",
        "description": "Invalid kind in catalog"
    }
}`,
			expected: ErrInvalidSchema,
		},
		{
			name:     "empty JSON data",
			jsonData: "",
			expected: ErrInvalidSchema,
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

	// Create the tenant for testing
	err := db.DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer db.DB(ctx).DeleteTenant(ctx, tenantID)

	// Create the project for testing
	err = db.DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {

			// Convert JSON to []byte
			jsonData := []byte(tt.jsonData)

			if tt.name == "invalid kind" {
				log.Println("err: ", err)
			}
			// Create a new catalog manager
			cm, err := NewCatalogManager(ctx, jsonData, "CatalogName")

			// Check if the error string matches the expected error string
			if !errors.Is(err, tt.expected) {
				t.Errorf("got error %v, expected error %v", err, tt.expected)
			} else if tt.expected == nil {
				if tt.name == "invalid kind" {
					log.Println("cm: ", cm)
				}
				// If no error is expected, validate catalog properties
				assert.NotNil(t, cm)
				assert.Equal(t, "valid-catalog", cm.Name())
				assert.Equal(t, "This is a valid catalog", cm.Description())

				// Save the catalog
				err = cm.Save(ctx)
				assert.NoError(t, err)

				// Attempt to save again to check for duplicate handling
				err = cm.Save(ctx)
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrAlreadyExists)

				// Load the catalog
				loadedCatalog, loadErr := LoadCatalogManagerByName(ctx, "valid-catalog")
				assert.NoError(t, loadErr)
				assert.Equal(t, cm.Name(), loadedCatalog.Name())
				assert.Equal(t, cm.Description(), loadedCatalog.Description())

				// Load the catalog with an invalid name
				_, loadErr = LoadCatalogManagerByName(ctx, "InvalidCatalog")
				assert.Error(t, loadErr)
				assert.ErrorIs(t, loadErr, ErrCatalogNotFound)

				// Delete the catalog
				err = DeleteCatalogByName(ctx, "ValidCatalog")
				assert.NoError(t, err)

				// Try loading the deleted catalog
				_, loadErr = LoadCatalogManagerByName(ctx, "ValidCatalog")
				assert.Error(t, loadErr)

				// Try Deleting again
				err = DeleteCatalogByName(ctx, "ValidCatalog")
				assert.NoError(t, err) // should not return an error
			}
		})
	}
}

func TestCatalogKindList(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := newDb()
	defer db.DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("PDEFGH")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant for testing
	err := db.DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer db.DB(ctx).DeleteTenant(ctx, tenantID)

	// Create the project for testing
	err = db.DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer db.DB(ctx).DeleteProject(ctx, projectID)

	// Create a single catalog for testing using the exact same format as the working test
	catalogJSON := `{
    "apiVersion": "0.1.0-alpha.1",
    "kind": "Catalog",
    "metadata": {
        "name": "test-catalog",
        "description": "Test catalog for listing"
    }
}`

	// Create the catalog
	cm, err := NewCatalogManager(ctx, []byte(catalogJSON), "")
	if err != nil {
		t.Logf("NewCatalogManager error: %v", err)
	}
	assert.NoError(t, err)
	assert.NotNil(t, cm)

	err = cm.Save(ctx)
	assert.NoError(t, err)
	defer DeleteCatalogByName(ctx, cm.Name())

	// Create a catalogKind handler
	reqContext := interfaces.RequestContext{
		Catalog: "test-catalog",
	}
	catalogKind, err := NewCatalogKindHandler(ctx, reqContext)
	assert.NoError(t, err)
	assert.NotNil(t, catalogKind)

	// Test the List method
	result, err := catalogKind.List(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Parse the JSON result
	var catalogNames []string
	err = json.Unmarshal(result, &catalogNames)
	assert.NoError(t, err)

	// Verify we got the expected catalog name
	assert.Len(t, catalogNames, 1)
	assert.Equal(t, "test-catalog", catalogNames[0])
}
