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
	"github.com/tansive/tansive/internal/common/uuid"
)

// The rules field here is garbage json and is immaterial.
func TestCreateView(t *testing.T) {
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

	var rules pgtype.JSONB
	assert.NoError(t, rules.Set(`{"filters": ["test"]}`))

	catalog := models.Catalog{
		Name:        "test_catalog_view",
		Description: "Catalog for view test",
		Info:        info,
	}
	require.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	view := models.View{
		Label:       "view_create_test",
		Description: "testing creation",
		Info:        info.Bytes,
		Rules:       rules.Bytes,
		CatalogID:   catalog.CatalogID,
		CreatedBy:   "test_user",
		UpdatedBy:   "test_user",
	}
	err := DB(ctx).CreateView(ctx, &view)
	assert.NoError(t, err)

	// Try duplicate create
	err = DB(ctx).CreateView(ctx, &view)
	assert.ErrorIs(t, err, dberror.ErrAlreadyExists)

	// Create new
	view.Label = "new-view"
	err = DB(ctx).CreateView(ctx, &view)
	assert.NoError(t, err)

	// Create with same label again
	err = DB(ctx).CreateView(ctx, &view)
	assert.Error(t, err)

	// Create with invalid label
	view.Label = "invalid label!" // invalid due to special character
	err = DB(ctx).CreateView(ctx, &view)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput, "should return ErrInvalidInput for invalid label")
}

func TestGetView(t *testing.T) {
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

	var rules pgtype.JSONB
	assert.NoError(t, rules.Set(`{"filters": ["test"]}`))

	catalog := models.Catalog{Name: "test_catalog", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	view := models.View{
		Label:       "view_get_test",
		Description: "desc",
		Info:        info.Bytes,
		Rules:       rules.Bytes,
		CatalogID:   catalog.CatalogID,
		CreatedBy:   "test_user",
		UpdatedBy:   "test_user",
	}
	assert.NoError(t, DB(ctx).CreateView(ctx, &view))

	// Positive case
	retrieved, err := DB(ctx).GetView(ctx, view.ViewID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, view.Label, retrieved.Label)
	assert.Equal(t, view.CatalogID, retrieved.CatalogID)
	assert.Equal(t, catalog.Name, retrieved.Catalog)

	// Negative case
	_, err = DB(ctx).GetView(ctx, view.ViewID)
	assert.NoError(t, err) // Should still find it
}

func TestGetViewByLabel(t *testing.T) {
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
	assert.NoError(t, info.Set(`{"meta": "get_by_label_test"}`))

	var rules pgtype.JSONB
	assert.NoError(t, rules.Set(`{"filters": ["test"]}`))

	catalog := models.Catalog{Name: "test_catalog", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	view := models.View{
		Label:       "view_get_by_label_test",
		Description: "desc",
		Info:        info.Bytes,
		Rules:       rules.Bytes,
		CatalogID:   catalog.CatalogID,
		CreatedBy:   "test_user",
		UpdatedBy:   "test_user",
	}
	assert.NoError(t, DB(ctx).CreateView(ctx, &view))

	// Positive case
	retrieved, err := DB(ctx).GetViewByLabel(ctx, view.Label, catalog.CatalogID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, view.Label, retrieved.Label)
	assert.Equal(t, view.CatalogID, retrieved.CatalogID)
	assert.Equal(t, catalog.Name, retrieved.Catalog)

	// Test getting non-existent view
	_, err = DB(ctx).GetViewByLabel(ctx, "non-existent", catalog.CatalogID)
	assert.Error(t, err)

	// Test getting view with wrong catalog ID
	_, err = DB(ctx).GetViewByLabel(ctx, view.Label, uuid.New())
	assert.Error(t, err)
}

func TestUpdateView(t *testing.T) {
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

	var rules pgtype.JSONB
	assert.NoError(t, rules.Set(`{"filters": ["test"]}`))

	catalog := models.Catalog{Name: "catalog_update", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	view := models.View{
		Label:       "view_update_test",
		Description: "old description",
		Info:        info.Bytes,
		Rules:       rules.Bytes,
		CatalogID:   catalog.CatalogID,
		CreatedBy:   "test_user",
		UpdatedBy:   "test_user",
	}
	assert.NoError(t, DB(ctx).CreateView(ctx, &view))

	// Update description
	view.Description = "new description"
	err := DB(ctx).UpdateView(ctx, &view)
	assert.NoError(t, err)

	retrieved, err := DB(ctx).GetView(ctx, view.ViewID)
	assert.NoError(t, err)
	assert.Equal(t, "new description", retrieved.Description)
	assert.Equal(t, "view_update_test", retrieved.Label) // Label should remain unchanged

	// Update info
	var newInfo pgtype.JSONB
	assert.NoError(t, newInfo.Set(`{"meta": "updated_info"}`))
	view.Info = newInfo.Bytes
	err = DB(ctx).UpdateView(ctx, &view)
	assert.NoError(t, err)

	retrieved, err = DB(ctx).GetView(ctx, view.ViewID)
	assert.NoError(t, err)
	assert.Equal(t, newInfo.Bytes, retrieved.Info)
	assert.Equal(t, "view_update_test", retrieved.Label) // Label should remain unchanged

	// Update rules
	var newRules pgtype.JSONB
	assert.NoError(t, newRules.Set(`{"filters": ["updated"]}`))
	view.Rules = newRules.Bytes
	err = DB(ctx).UpdateView(ctx, &view)
	assert.NoError(t, err)

	retrieved, err = DB(ctx).GetView(ctx, view.ViewID)
	assert.NoError(t, err)
	assert.Equal(t, newRules.Bytes, retrieved.Rules)
	assert.Equal(t, "view_update_test", retrieved.Label) // Label should remain unchanged

	// Verify that label changes are ignored
	view.Label = "attempted_label_change"
	err = DB(ctx).UpdateView(ctx, &view)
	assert.NoError(t, err)

	retrieved, err = DB(ctx).GetView(ctx, view.ViewID)
	assert.NoError(t, err)
	assert.Equal(t, "view_update_test", retrieved.Label) // Label should remain unchanged despite update attempt
}

func TestDeleteView(t *testing.T) {
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

	var rules pgtype.JSONB
	assert.NoError(t, rules.Set(`{"filters": ["test"]}`))

	catalog := models.Catalog{Name: "catalog_delete", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	view := models.View{
		Label:       "view_delete_test",
		Description: "desc",
		Info:        info.Bytes,
		Rules:       rules.Bytes,
		CatalogID:   catalog.CatalogID,
		CreatedBy:   "test_user",
		UpdatedBy:   "test_user",
	}
	assert.NoError(t, DB(ctx).CreateView(ctx, &view))

	// Delete
	err := DB(ctx).DeleteView(ctx, view.ViewID)
	assert.NoError(t, err)

	// Verify
	_, err = DB(ctx).GetView(ctx, view.ViewID)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Delete again (should fail as not found)
	err = DB(ctx).DeleteView(ctx, view.ViewID)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestListViewsByCatalog(t *testing.T) {
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

	var rules pgtype.JSONB
	assert.NoError(t, rules.Set(`{"filters": ["test"]}`))

	catalog := models.Catalog{Name: "catalog_list", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	// Create multiple views
	views := []models.View{
		{
			Label:     "view1",
			Info:      info.Bytes,
			Rules:     rules.Bytes,
			CatalogID: catalog.CatalogID,
			CreatedBy: "test_user",
			UpdatedBy: "test_user",
		},
		{
			Label:     "view2",
			Info:      info.Bytes,
			Rules:     rules.Bytes,
			CatalogID: catalog.CatalogID,
			CreatedBy: "test_user",
			UpdatedBy: "test_user",
		},
		{
			Label:     "view3",
			Info:      info.Bytes,
			Rules:     rules.Bytes,
			CatalogID: catalog.CatalogID,
			CreatedBy: "test_user",
			UpdatedBy: "test_user",
		},
	}

	for i := range views {
		assert.NoError(t, DB(ctx).CreateView(ctx, &views[i]))
	}

	// List views
	retrieved, err := DB(ctx).ListViewsByCatalog(ctx, catalog.CatalogID)
	assert.NoError(t, err)
	require.Len(t, retrieved, 4) // there's already one default view created

	// Verify order (should be alphabetical by label)
	assert.Equal(t, "view1", retrieved[1].Label)
	assert.Equal(t, "view2", retrieved[2].Label)
	assert.Equal(t, "view3", retrieved[3].Label)
}
