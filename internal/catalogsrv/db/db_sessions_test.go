package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestCreateSession(t *testing.T) {
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

	var status pgtype.JSONB
	assert.NoError(t, status.Set(`{"state": "active"}`))

	var viewDef pgtype.JSONB
	assert.NoError(t, viewDef.Set(`{"view": "test"}`))

	var variables pgtype.JSONB
	assert.NoError(t, variables.Set(`{"var1": "value1", "var2": 123}`))

	catalog := models.Catalog{
		Name:        "test_catalog_session",
		Description: "Catalog for session test",
		Info:        info,
	}
	require.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{
		Name:        "test_variant",
		Description: "Variant for session test",
		Info:        info,
		CatalogID:   catalog.CatalogID,
	}
	require.NoError(t, DB(ctx).CreateVariant(ctx, &variant))

	view := models.View{
		Label:       "test_view",
		Description: "View for session test",
		Info:        info.Bytes,
		Rules:       viewDef.Bytes,
		CatalogID:   catalog.CatalogID,
		CreatedBy:   "test_user",
		UpdatedBy:   "test_user",
	}
	require.NoError(t, DB(ctx).CreateView(ctx, &view))

	session := models.Session{
		SessionID: uuid.New(),
		SkillSet:  "test_skillset",
		Skill:     "test_skill",
		ViewID:    view.ViewID,
		TangentID: uuid.New(),
		Status:    nil,
		Info:      nil,
		UserID:    "test_user",
		CatalogID: catalog.CatalogID,
		VariantID: variant.VariantID,
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(time.Hour),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := DB(ctx).UpsertSession(ctx, &session)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, session.SessionID)
}

func TestGetSession(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	db := DB(ctx)

	assert.NoError(t, db.CreateTenant(ctx, tenantID))
	defer db.DeleteTenant(ctx, tenantID)

	assert.NoError(t, db.CreateProject(ctx, projectID))
	defer db.DeleteProject(ctx, projectID)

	// Common JSON metadata
	rawInfo := json.RawMessage(`{"meta": "get_test"}`)
	rawStatus := json.RawMessage(`{"state": "active"}`)
	rawViewDef := json.RawMessage(`{"view": "test"}`)

	var info pgtype.JSONB
	require.NoError(t, info.Set(`{"meta": "get_test"}`))

	catalog := models.Catalog{
		Name: "test_catalog",
		Info: info,
	}
	require.NoError(t, db.CreateCatalog(ctx, &catalog))
	defer db.DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{
		Name:      "test_variant",
		Info:      info,
		CatalogID: catalog.CatalogID,
	}
	require.NoError(t, db.CreateVariant(ctx, &variant))

	view := models.View{
		Label:     "test_view",
		Info:      rawInfo,
		Rules:     rawViewDef,
		CatalogID: catalog.CatalogID,
		CreatedBy: "test_user",
		UpdatedBy: "test_user",
	}
	require.NoError(t, db.CreateView(ctx, &view))

	session := models.Session{
		SkillSet:  "test_skillset",
		Skill:     "test_skill",
		ViewID:    view.ViewID,
		TangentID: uuid.New(),
		Status:    rawStatus,
		Info:      rawInfo,
		UserID:    "test_user",
		CatalogID: catalog.CatalogID,
		VariantID: variant.VariantID,
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(time.Hour),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	assert.NoError(t, db.UpsertSession(ctx, &session))

	// Positive case
	retrieved, err := db.GetSession(ctx, session.SessionID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, session.SessionID, retrieved.SessionID)
	assert.Equal(t, session.SkillSet, retrieved.SkillSet)
	assert.Equal(t, session.Skill, retrieved.Skill)
	assert.Equal(t, session.ViewID, retrieved.ViewID)

	// Compare RawMessage fields (normalize via JSON strings)
	assert.JSONEq(t, string(session.Info), string(retrieved.Info))
	assert.JSONEq(t, string(session.Status), string(retrieved.Status))

	// Negative case
	_, err = db.GetSession(ctx, uuid.New())
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestUpdateSessionStatus(t *testing.T) {
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

	var status pgtype.JSONB
	assert.NoError(t, status.Set(`{"state": "active"}`))

	var viewDef pgtype.JSONB
	assert.NoError(t, viewDef.Set(`{"view": "test"}`))

	var variables pgtype.JSONB
	assert.NoError(t, variables.Set(`{"var1": "value1", "var2": 123}`))

	catalog := models.Catalog{Name: "test_catalog", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{
		Name:      "test_variant",
		Info:      info,
		CatalogID: catalog.CatalogID,
	}
	require.NoError(t, DB(ctx).CreateVariant(ctx, &variant))

	view := models.View{
		Label:     "test_view",
		Info:      info.Bytes,
		Rules:     viewDef.Bytes,
		CatalogID: catalog.CatalogID,
		CreatedBy: "test_user",
		UpdatedBy: "test_user",
	}
	require.NoError(t, DB(ctx).CreateView(ctx, &view))

	session := models.Session{
		SkillSet:  "test_skillset",
		Skill:     "test_skill",
		ViewID:    view.ViewID,
		TangentID: uuid.New(),
		Status:    status.Bytes,
		Info:      info.Bytes,
		UserID:    "test_user",
		CatalogID: catalog.CatalogID,
		VariantID: variant.VariantID,
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(time.Hour),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	assert.NoError(t, DB(ctx).UpsertSession(ctx, &session))

	// Update status
	var newStatus pgtype.JSONB
	assert.NoError(t, newStatus.Set(`{"state": "completed"}`))
	err := DB(ctx).UpdateSessionStatus(ctx, session.SessionID, "completed", newStatus.Bytes)
	assert.NoError(t, err)

	retrieved, err := DB(ctx).GetSession(ctx, session.SessionID)
	assert.NoError(t, err)
	assert.JSONEq(t, string(newStatus.Bytes), string(retrieved.Status))
	assert.Equal(t, "completed", retrieved.StatusSummary)

	// Update non-existent session
	err = DB(ctx).UpdateSessionStatus(ctx, uuid.New(), "completed", newStatus.Bytes)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestUpdateSessionEnd(t *testing.T) {
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
	assert.NoError(t, info.Set(`{"meta": "end_test"}`))

	var status pgtype.JSONB
	assert.NoError(t, status.Set(`{"state": "active"}`))

	var viewDef pgtype.JSONB
	assert.NoError(t, viewDef.Set(`{"view": "test"}`))

	var variables pgtype.JSONB
	assert.NoError(t, variables.Set(`{"var1": "value1", "var2": 123}`))

	catalog := models.Catalog{Name: "test_catalog", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{
		Name:      "test_variant",
		Info:      info,
		CatalogID: catalog.CatalogID,
	}
	require.NoError(t, DB(ctx).CreateVariant(ctx, &variant))

	view := models.View{
		Label:     "test_view",
		Info:      info.Bytes,
		Rules:     viewDef.Bytes,
		CatalogID: catalog.CatalogID,
		CreatedBy: "test_user",
		UpdatedBy: "test_user",
	}
	require.NoError(t, DB(ctx).CreateView(ctx, &view))

	session := models.Session{
		SkillSet:  "test_skillset",
		Skill:     "test_skill",
		ViewID:    view.ViewID,
		TangentID: uuid.New(),
		Status:    status.Bytes,
		Info:      info.Bytes,
		UserID:    "test_user",
		CatalogID: catalog.CatalogID,
		VariantID: variant.VariantID,
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(time.Hour),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	assert.NoError(t, DB(ctx).UpsertSession(ctx, &session))

	// Update status and end session
	var newStatus pgtype.JSONB
	assert.NoError(t, newStatus.Set(`{"state": "completed"}`))
	err := DB(ctx).UpdateSessionEnd(ctx, session.SessionID, "completed", newStatus.Bytes)
	assert.NoError(t, err)

	retrieved, err := DB(ctx).GetSession(ctx, session.SessionID)
	assert.NoError(t, err)
	assert.JSONEq(t, string(newStatus.Bytes), string(retrieved.Status))
	assert.Equal(t, "completed", retrieved.StatusSummary)
	assert.True(t, retrieved.EndedAt.Before(time.Now().Add(time.Second)))
	assert.True(t, retrieved.EndedAt.After(time.Now().Add(-time.Second)))

	// Update non-existent session
	err = DB(ctx).UpdateSessionEnd(ctx, uuid.New(), "completed", newStatus.Bytes)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestUpdateSessionInfo(t *testing.T) {
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
	assert.NoError(t, info.Set(`{"meta": "info_test"}`))

	var status pgtype.JSONB
	assert.NoError(t, status.Set(`{"state": "active"}`))

	var viewDef pgtype.JSONB
	assert.NoError(t, viewDef.Set(`{"view": "test"}`))

	var variables pgtype.JSONB
	assert.NoError(t, variables.Set(`{"var1": "value1", "var2": 123}`))

	catalog := models.Catalog{Name: "test_catalog", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{
		Name:      "test_variant",
		Info:      info,
		CatalogID: catalog.CatalogID,
	}
	require.NoError(t, DB(ctx).CreateVariant(ctx, &variant))

	view := models.View{
		Label:     "test_view",
		Info:      info.Bytes,
		Rules:     viewDef.Bytes,
		CatalogID: catalog.CatalogID,
		CreatedBy: "test_user",
		UpdatedBy: "test_user",
	}
	require.NoError(t, DB(ctx).CreateView(ctx, &view))

	session := models.Session{
		SkillSet:  "test_skillset",
		Skill:     "test_skill",
		ViewID:    view.ViewID,
		TangentID: uuid.New(),
		Status:    status.Bytes,
		Info:      info.Bytes,
		UserID:    "test_user",
		CatalogID: catalog.CatalogID,
		VariantID: variant.VariantID,
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(time.Hour),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	assert.NoError(t, DB(ctx).UpsertSession(ctx, &session))

	// Update info
	var newInfo pgtype.JSONB
	assert.NoError(t, newInfo.Set(`{"meta": "updated_info"}`))
	err := DB(ctx).UpdateSessionInfo(ctx, session.SessionID, newInfo.Bytes)
	assert.NoError(t, err)

	retrieved, err := DB(ctx).GetSession(ctx, session.SessionID)
	assert.NoError(t, err)
	assert.JSONEq(t, string(newInfo.Bytes), string(retrieved.Info))

	// Update non-existent session
	err = DB(ctx).UpdateSessionInfo(ctx, uuid.New(), newInfo.Bytes)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestDeleteSession(t *testing.T) {
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

	var status pgtype.JSONB
	assert.NoError(t, status.Set(`{"state": "active"}`))

	var viewDef pgtype.JSONB
	assert.NoError(t, viewDef.Set(`{"view": "test"}`))

	var variables pgtype.JSONB
	assert.NoError(t, variables.Set(`{"var1": "value1", "var2": 123}`))

	catalog := models.Catalog{Name: "test_catalog", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{
		Name:      "test_variant",
		Info:      info,
		CatalogID: catalog.CatalogID,
	}
	require.NoError(t, DB(ctx).CreateVariant(ctx, &variant))

	view := models.View{
		Label:     "test_view",
		Info:      info.Bytes,
		Rules:     viewDef.Bytes,
		CatalogID: catalog.CatalogID,
		CreatedBy: "test_user",
		UpdatedBy: "test_user",
	}
	require.NoError(t, DB(ctx).CreateView(ctx, &view))

	session := models.Session{
		SkillSet:  "test_skillset",
		Skill:     "test_skill",
		ViewID:    view.ViewID,
		TangentID: uuid.New(),
		Status:    status.Bytes,
		Info:      info.Bytes,
		UserID:    "test_user",
		CatalogID: catalog.CatalogID,
		VariantID: variant.VariantID,
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(time.Hour),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	assert.NoError(t, DB(ctx).UpsertSession(ctx, &session))

	// Delete
	err := DB(ctx).DeleteSession(ctx, session.SessionID)
	assert.NoError(t, err)

	// Verify
	_, err = DB(ctx).GetSession(ctx, session.SessionID)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Delete again (should fail as not found)
	err = DB(ctx).DeleteSession(ctx, session.SessionID)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestListSessionsByCatalog(t *testing.T) {
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

	var status pgtype.JSONB
	assert.NoError(t, status.Set(`{"state": "active"}`))

	var viewDef pgtype.JSONB
	assert.NoError(t, viewDef.Set(`{"view": "test"}`))

	var variables pgtype.JSONB
	assert.NoError(t, variables.Set(`{"var1": "value1", "var2": 123}`))

	catalog := models.Catalog{Name: "test_catalog", Info: info}
	assert.NoError(t, DB(ctx).CreateCatalog(ctx, &catalog))
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant := models.Variant{
		Name:      "test_variant",
		Info:      info,
		CatalogID: catalog.CatalogID,
	}
	require.NoError(t, DB(ctx).CreateVariant(ctx, &variant))

	view := models.View{
		Label:     "test_view",
		Info:      info.Bytes,
		Rules:     viewDef.Bytes,
		CatalogID: catalog.CatalogID,
		CreatedBy: "test_user",
		UpdatedBy: "test_user",
	}
	require.NoError(t, DB(ctx).CreateView(ctx, &view))

	// Create multiple sessions
	sessions := []models.Session{
		{
			SessionID: uuid.New(),
			SkillSet:  "skillset1",
			Skill:     "skill1",
			ViewID:    view.ViewID,
			TangentID: uuid.New(),
			Status:    status.Bytes,
			Info:      info.Bytes,
			UserID:    "test_user",
			CatalogID: catalog.CatalogID,
			VariantID: variant.VariantID,
			StartedAt: time.Now(),
			EndedAt:   time.Now().Add(time.Hour),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
		{
			SessionID: uuid.New(),
			SkillSet:  "skillset2",
			Skill:     "skill2",
			ViewID:    view.ViewID,
			TangentID: uuid.New(),
			Status:    status.Bytes,
			Info:      info.Bytes,
			UserID:    "test_user",
			CatalogID: catalog.CatalogID,
			VariantID: variant.VariantID,
			StartedAt: time.Now(),
			EndedAt:   time.Now().Add(time.Hour),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
		{
			SessionID: uuid.New(),
			SkillSet:  "skillset3",
			Skill:     "skill3",
			ViewID:    view.ViewID,
			TangentID: uuid.New(),
			Status:    status.Bytes,
			Info:      info.Bytes,
			UserID:    "test_user",
			CatalogID: catalog.CatalogID,
			VariantID: variant.VariantID,
			StartedAt: time.Now(),
			EndedAt:   time.Now().Add(time.Hour),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}

	for i := range sessions {
		assert.NoError(t, DB(ctx).UpsertSession(ctx, &sessions[i]))
	}

	// List sessions
	retrieved, err := DB(ctx).ListSessionsByCatalog(ctx, catalog.CatalogID)
	assert.NoError(t, err)
	require.Len(t, retrieved, 3)

	// Verify order (should be by created_at DESC)
	assert.True(t, retrieved[0].CreatedAt.After(retrieved[1].CreatedAt))
	assert.True(t, retrieved[1].CreatedAt.After(retrieved[2].CreatedAt))
}
