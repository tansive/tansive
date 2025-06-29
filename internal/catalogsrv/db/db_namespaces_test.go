package db

import (
	"context"
	"testing"

	"github.com/jackc/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
)

func TestCreateNamespace(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	assert.NoError(t, DB(ctx).CreateTenant(ctx, tenantID))
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	assert.NoError(t, DB(ctx).CreateProject(ctx, projectID))
	defer DB(ctx).DeleteProject(ctx, projectID)

	var info pgtype.JSONB
	assert.NoError(t, info.Set(`{"meta": "test"}`))

	catalog := models.Catalog{
		Name:        "test_catalog_ns",
		Description: "Catalog for namespace test",
		Info:        info,
	}
	require.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{
		Name:        "test_variant_ns",
		Description: "Variant for namespace test",
		CatalogID:   catalog.CatalogID,
		Info:        info,
	}
	assert.NoError(t, DB(ctx).CreateVariant(ctx, &variant))
	defer DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")

	ns := models.Namespace{
		Name:        "ns_create_test",
		VariantID:   variant.VariantID,
		Description: "testing creation",
		Info:        info.Bytes,
	}
	err := DB(ctx).CreateNamespace(ctx, &ns)
	assert.NoError(t, err)

	// Try duplicate create
	err = DB(ctx).CreateNamespace(ctx, &ns)
	assert.ErrorIs(t, err, dberror.ErrAlreadyExists)

	// Create new
	ns.Name = "new-namespace"
	err = DB(ctx).CreateNamespace(ctx, &ns)
	assert.NoError(t, err)

	// Create with same name again
	err = DB(ctx).CreateNamespace(ctx, &ns)
	assert.Error(t, err)

	// Create with invalid name
	ns.Name = "invalid name!" // invalid due to special character
	err = DB(ctx).CreateNamespace(ctx, &ns)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput, "should return ErrInvalidInput for invalid name")
}

func TestGetNamespace(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	assert.NoError(t, DB(ctx).CreateTenant(ctx, tenantID))
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	assert.NoError(t, DB(ctx).CreateProject(ctx, projectID))
	defer DB(ctx).DeleteProject(ctx, projectID)

	var info pgtype.JSONB
	assert.NoError(t, info.Set(`{"meta": "get_test"}`))

	catalog := models.Catalog{Name: "test_catalog", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{Name: "test_variant", CatalogID: catalog.CatalogID, Info: info}
	assert.NoError(t, DB(ctx).CreateVariant(ctx, &variant))
	defer DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")

	ns := models.Namespace{
		Name:        "ns_get_test",
		VariantID:   variant.VariantID,
		Description: "desc",
		Info:        info.Bytes,
	}
	assert.NoError(t, DB(ctx).CreateNamespace(ctx, &ns))

	// Positive case
	retrieved, err := DB(ctx).GetNamespace(ctx, ns.Name, ns.VariantID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, ns.Name, retrieved.Name)

	// Negative case
	_, err = DB(ctx).GetNamespace(ctx, "does_not_exist", ns.VariantID)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestUpdateNamespace(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	assert.NoError(t, DB(ctx).CreateTenant(ctx, tenantID))
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	assert.NoError(t, DB(ctx).CreateProject(ctx, projectID))
	defer DB(ctx).DeleteProject(ctx, projectID)

	var info pgtype.JSONB
	assert.NoError(t, info.Set(`{"meta": "update_test"}`))

	catalog := models.Catalog{Name: "catalog_update", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{Name: "variant_update", CatalogID: catalog.CatalogID, Info: info}
	assert.NoError(t, DB(ctx).CreateVariant(ctx, &variant))
	defer DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")

	ns := models.Namespace{
		Name:        "ns_update_test",
		VariantID:   variant.VariantID,
		Description: "old description",
		Info:        info.Bytes,
	}
	assert.NoError(t, DB(ctx).CreateNamespace(ctx, &ns))

	// Update description
	ns.Description = "new description"
	err := DB(ctx).UpdateNamespace(ctx, &ns)
	assert.NoError(t, err)

	retrieved, err := DB(ctx).GetNamespace(ctx, ns.Name, ns.VariantID)
	assert.NoError(t, err)
	assert.Equal(t, "new description", retrieved.Description)

	// Nonexistent
	ns.Name = "nonexistent"
	err = DB(ctx).UpdateNamespace(ctx, &ns)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestDeleteNamespace(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	assert.NoError(t, DB(ctx).CreateTenant(ctx, tenantID))
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	assert.NoError(t, DB(ctx).CreateProject(ctx, projectID))
	defer DB(ctx).DeleteProject(ctx, projectID)

	var info pgtype.JSONB
	assert.NoError(t, info.Set(`{"meta": "delete_test"}`))

	catalog := models.Catalog{Name: "catalog_delete", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{Name: "variant_delete", CatalogID: catalog.CatalogID, Info: info}
	assert.NoError(t, DB(ctx).CreateVariant(ctx, &variant))
	defer DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")

	ns := models.Namespace{
		Name:        "ns_delete_test",
		VariantID:   variant.VariantID,
		Description: "desc",
		Info:        info.Bytes,
	}
	assert.NoError(t, DB(ctx).CreateNamespace(ctx, &ns))

	// Delete
	err := DB(ctx).DeleteNamespace(ctx, ns.Name, ns.VariantID)
	assert.NoError(t, err)

	// Verify
	_, err = DB(ctx).GetNamespace(ctx, ns.Name, ns.VariantID)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Delete again (should fail as not found)
	err = DB(ctx).DeleteNamespace(ctx, ns.Name, ns.VariantID)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestListNamespacesByVariant(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	assert.NoError(t, DB(ctx).CreateTenant(ctx, tenantID))
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	assert.NoError(t, DB(ctx).CreateProject(ctx, projectID))
	defer DB(ctx).DeleteProject(ctx, projectID)

	var info pgtype.JSONB
	assert.NoError(t, info.Set(`{"meta": "list_test"}`))

	catalog := models.Catalog{Name: "catalog_list", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{Name: "variant_list", CatalogID: catalog.CatalogID, Info: info}
	assert.NoError(t, DB(ctx).CreateVariant(ctx, &variant))
	defer DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")

	// Create multiple namespaces
	for _, name := range []string{"ns1", "ns2", "ns3"} {
		ns := models.Namespace{
			Name:        name,
			VariantID:   variant.VariantID,
			Description: "desc " + name,
			Info:        info.Bytes,
		}
		assert.NoError(t, DB(ctx).CreateNamespace(ctx, &ns))
	}

	// List
	namespaces, err := DB(ctx).ListNamespacesByVariant(ctx, variant.VariantID)
	assert.NoError(t, err)
	assert.Len(t, namespaces, 4)

	// Delete all created namespaces to clean up
	for _, ns := range namespaces {
		err := DB(ctx).DeleteNamespace(ctx, ns.Name, ns.VariantID)
		assert.NoError(t, err, "failed to delete namespace %s: %v", ns.Name, err)
	}
	// Verify deletion
	for _, ns := range namespaces {
		_, err := DB(ctx).GetNamespace(ctx, ns.Name, ns.VariantID)
		assert.ErrorIs(t, err, dberror.ErrNotFound, "namespace %s should be deleted", ns.Name)
	}
}
