package db

import (
	"context"
	"testing"

	"github.com/jackc/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestCreateTenant(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")

	// Test successful tenant creation
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	// Test trying to create the same tenant again (should return ErrAlreadyExists)
	err = DB(ctx).CreateTenant(ctx, tenantID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrAlreadyExists)
}

func TestGetTenant(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	// First, create the tenant to test retrieval
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)

	// Test successfully retrieving the created tenant
	tenant, err := DB(ctx).GetTenant(ctx, tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.Equal(t, tenantID, tenant.TenantID)

	// Test trying to get a non-existent tenant (should return ErrNotFound)
	nonExistentTenantID := catcommon.TenantId("nonexistent123")
	tenant, err = DB(ctx).GetTenant(ctx, nonExistentTenantID)
	assert.Error(t, err)
	assert.Nil(t, tenant)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestCreateProject(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)

	// Create the tenant to associate with the project
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	// Test creating a new project
	err = DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Test trying to create the same project (should return ErrAlreadyExists)
	err = DB(ctx).CreateProject(ctx, projectID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrAlreadyExists)

	// Test trying to create a project without a tenant ID in the context
	ctx = catcommon.WithTenantID(ctx, "")
	err = DB(ctx).CreateProject(ctx, projectID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrDatabase)
}

func TestGetProject(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)

	// Create the tenant and project to test retrieval
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Test successfully retrieving the created project
	project, err := DB(ctx).GetProject(ctx, projectID)
	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, projectID, project.ProjectID)
	assert.Equal(t, tenantID, project.TenantID)

	// Test trying to get a non-existent project (should return ErrNotFound)
	nonExistentProjectID := catcommon.ProjectId("nonexistent123")
	project, err = DB(ctx).GetProject(ctx, nonExistentProjectID)
	assert.Error(t, err)
	assert.Nil(t, project)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestDeleteProject(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)

	// Create the tenant and project to test deletion
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Test deleting the project
	err = DB(ctx).DeleteProject(ctx, projectID)
	assert.NoError(t, err)

	// Test trying to delete a non-existent project (should succeed without errors)
	err = DB(ctx).DeleteProject(ctx, projectID)
	assert.NoError(t, err)

	// Test trying to retrieve a deleted project (should return ErrNotFound)
	project, err := DB(ctx).GetProject(ctx, projectID)
	assert.Error(t, err)
	assert.Nil(t, project)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestCreateCatalog(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant and project for testing
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog with non-empty info
	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	assert.NoError(t, err)

	catalog := &models.Catalog{
		Name:        "test_catalog",
		Description: "A test catalog",
		Info:        info,
	}
	err = DB(ctx).CreateCatalog(ctx, catalog)
	assert.NoError(t, err)
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	// Try to get id of created catalog by name
	cId, err := DB(ctx).GetCatalogIDByName(ctx, catalog.Name)
	assert.NoError(t, err)
	assert.Equal(t, catalog.CatalogID, cId)

	// Try to get an invalid name
	cId, err = DB(ctx).GetCatalogIDByName(ctx, "invalid_name")
	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, cId)

	// Try to create a catalog with the same name, project, and tenant (should return ErrAlreadyExists)
	err = DB(ctx).CreateCatalog(ctx, catalog)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrAlreadyExists)

	// Create a catalog with empty info
	var emptyInfo pgtype.JSONB
	err = emptyInfo.Set(`{}`)
	assert.NoError(t, err)

	emptyCatalog := models.Catalog{
		Name:        "test_catalog_empty",
		Description: "A test catalog with empty info",
		Info:        emptyInfo,
	}
	err = DB(ctx).CreateCatalog(ctx, &emptyCatalog)
	assert.NoError(t, err)
	defer DB(ctx).DeleteCatalog(ctx, emptyCatalog.CatalogID, "")
}

func TestGetCatalog(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant and project to test retrieval
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog for retrieval
	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	assert.NoError(t, err)

	catalog := models.Catalog{
		Name:        "test_catalog",
		Description: "A test catalog",
		Info:        info,
	}
	err = DB(ctx).CreateCatalog(ctx, &catalog)
	assert.NoError(t, err)
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	// Test successfully retrieving the created catalog by catalogID
	retrievedCatalog, err := DB(ctx).GetCatalogByID(ctx, catalog.CatalogID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedCatalog)
	assert.Equal(t, catalog.Name, retrievedCatalog.Name)
	assert.Equal(t, catalog.Description, retrievedCatalog.Description)

	// Test successfully retrieving the created catalog by name
	retrievedCatalog, err = DB(ctx).GetCatalogByName(ctx, catalog.Name)
	if assert.NoError(t, err) {
		assert.NotNil(t, retrievedCatalog)
		assert.Equal(t, catalog.Name, retrievedCatalog.Name)
		assert.Equal(t, catalog.Description, retrievedCatalog.Description)
	}

	// Test trying to get a non-existent catalog by ID (should return ErrNotFound)
	nonExistentCatalogID := uuid.New()
	retrievedCatalog, err = DB(ctx).GetCatalogByID(ctx, nonExistentCatalogID)
	assert.Error(t, err)
	assert.Nil(t, retrievedCatalog)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Test trying to get a non-existent catalog by name (should return ErrNotFound)
	retrievedCatalog, err = DB(ctx).GetCatalogByName(ctx, "non_existent_catalog")
	assert.Error(t, err)
	assert.Nil(t, retrievedCatalog)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestUpdateCatalog(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant and project for testing
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog for updating
	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	assert.NoError(t, err)

	catalog := models.Catalog{
		Name:        "test_catalog",
		Description: "A test catalog",
		Info:        info,
	}
	err = DB(ctx).CreateCatalog(ctx, &catalog)
	assert.NoError(t, err)
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	// Update the catalog
	catalog.Description = "An updated description"
	err = DB(ctx).UpdateCatalog(ctx, &catalog)
	assert.NoError(t, err)

	// Retrieve the updated catalog and verify the changes
	retrievedCatalog, err := DB(ctx).GetCatalogByID(ctx, catalog.CatalogID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedCatalog)
	assert.Equal(t, "An updated description", retrievedCatalog.Description)

	// Test invalid input: neither catalogID nor name is provided
	invalidCatalog := models.Catalog{
		Description: "Should fail",
	}
	err = DB(ctx).UpdateCatalog(ctx, &invalidCatalog)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)

	// Test trying to update a non-existent catalog (should return ErrNotFound)
	nonExistentCatalog := models.Catalog{
		CatalogID:   uuid.New(),
		Name:        "non_existent_catalog",
		Description: "This catalog does not exist",
		Info:        info,
	}
	err = DB(ctx).UpdateCatalog(ctx, &nonExistentCatalog)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestDeleteCatalog(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant and project for testing
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Create a catalog for deletion
	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	assert.NoError(t, err)

	catalog := models.Catalog{
		Name:        "test_catalog",
		Description: "A test catalog",
		Info:        info,
	}
	err = DB(ctx).CreateCatalog(ctx, &catalog)
	assert.NoError(t, err)

	// Delete the catalog by catalogID
	err = DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")
	assert.NoError(t, err)

	// Try to retrieve the deleted catalog (should return ErrNotFound)
	retrievedCatalog, err := DB(ctx).GetCatalogByID(ctx, catalog.CatalogID)
	assert.Error(t, err)
	assert.Nil(t, retrievedCatalog)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Create the catalog again for deletion by name
	err = DB(ctx).CreateCatalog(ctx, &catalog)
	assert.NoError(t, err)

	// Delete the catalog by name
	err = DB(ctx).DeleteCatalog(ctx, uuid.Nil, catalog.Name)
	assert.NoError(t, err)

	// Try to retrieve the deleted catalog (should return ErrNotFound)
	retrievedCatalog, err = DB(ctx).GetCatalogByName(ctx, catalog.Name)
	assert.Error(t, err)
	assert.Nil(t, retrievedCatalog)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Test invalid input: neither catalogID nor name is provided
	err = DB(ctx).DeleteCatalog(ctx, uuid.Nil, "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)
}

func TestListCatalogs(t *testing.T) {
	// Initialize context with logger and database connection
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")

	// Set the tenant ID and project ID in the context
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create the tenant and project for testing
	err := DB(ctx).CreateTenant(ctx, tenantID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	assert.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Create multiple catalogs for testing
	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	assert.NoError(t, err)

	catalogs := []models.Catalog{
		{
			Name:        "catalog_a",
			Description: "First test catalog",
			Info:        info,
		},
		{
			Name:        "catalog_b",
			Description: "Second test catalog",
			Info:        info,
		},
		{
			Name:        "catalog_c",
			Description: "Third test catalog",
			Info:        info,
		},
	}

	// Create the catalogs
	for i := range catalogs {
		err = DB(ctx).CreateCatalog(ctx, &catalogs[i])
		assert.NoError(t, err)
		defer DB(ctx).DeleteCatalog(ctx, catalogs[i].CatalogID, "")
	}

	// List all catalogs
	retrievedCatalogs, err := DB(ctx).ListCatalogs(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedCatalogs)
	assert.Len(t, retrievedCatalogs, 3)

	// Verify the catalogs are returned in alphabetical order by name
	assert.Equal(t, "catalog_a", retrievedCatalogs[0].Name)
	assert.Equal(t, "catalog_b", retrievedCatalogs[1].Name)
	assert.Equal(t, "catalog_c", retrievedCatalogs[2].Name)

	// Verify the descriptions are correct
	assert.Equal(t, "First test catalog", retrievedCatalogs[0].Description)
	assert.Equal(t, "Second test catalog", retrievedCatalogs[1].Description)
	assert.Equal(t, "Third test catalog", retrievedCatalogs[2].Description)

	// Test with missing tenant ID
	ctxWithoutTenant := log.Logger.WithContext(context.Background())
	ctxWithoutTenant = newDb(ctxWithoutTenant)
	defer DB(ctxWithoutTenant).Close(ctxWithoutTenant)
	ctxWithoutTenant = catcommon.WithProjectID(ctxWithoutTenant, projectID)

	_, err = DB(ctxWithoutTenant).ListCatalogs(ctxWithoutTenant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrMissingTenantID)

	// Test with missing project ID
	ctxWithoutProject := log.Logger.WithContext(context.Background())
	ctxWithoutProject = newDb(ctxWithoutProject)
	defer DB(ctxWithoutProject).Close(ctxWithoutProject)
	ctxWithoutProject = catcommon.WithTenantID(ctxWithoutProject, tenantID)

	_, err = DB(ctxWithoutProject).ListCatalogs(ctxWithoutProject)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)
}

func newDb(c ...context.Context) context.Context {
	config.TestInit()
	Init()
	var ctx context.Context
	var err error
	if len(c) > 0 {
		ctx, err = ConnCtx(c[0])
		if err != nil {
			log.Ctx(ctx).Fatal().Err(err).Msg("unable to get db connection")
		}
	} else {
		ctx, err = ConnCtx(context.Background())
		if err != nil {
			log.Ctx(ctx).Fatal().Err(err).Msg("unable to get db connection")
		}
	}
	ctx = catcommon.WithCatalogContext(ctx, &catcommon.CatalogContext{
		UserContext: &catcommon.UserContext{
			UserID: "user/test_user",
		},
	})
	return ctx
}
