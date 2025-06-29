package db

import (
	"context"
	"encoding/json"
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

func TestCreateTangent(t *testing.T) {
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

	// Create first tangent
	tangent := models.Tangent{
		ID:        uuid.New(),
		Info:      info.Bytes,
		PublicKey: []byte("test-public-key-1"),
		Status:    "active",
	}
	err := DB(ctx).CreateTangent(ctx, &tangent)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tangent.ID)

	// Try to create another tangent with the same ID
	duplicateTangent := models.Tangent{
		ID:        tangent.ID, // Use the same ID
		Info:      info.Bytes,
		PublicKey: []byte("test-public-key-2"),
		Status:    "pending",
	}
	err = DB(ctx).CreateTangent(ctx, &duplicateTangent)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrAlreadyExists)

	// Create new tangent with different ID
	newTangent := models.Tangent{
		ID:        uuid.New(),
		Info:      info.Bytes,
		PublicKey: []byte("test-public-key-3"),
		Status:    "pending",
	}
	err = DB(ctx).CreateTangent(ctx, &newTangent)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, newTangent.ID)
	assert.NotEqual(t, tangent.ID, newTangent.ID)
}

func TestGetTangent(t *testing.T) {
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

	tangent := models.Tangent{
		ID:        uuid.New(),
		Info:      info.Bytes,
		PublicKey: []byte("test-public-key-get"),
		Status:    "active",
	}
	assert.NoError(t, DB(ctx).CreateTangent(ctx, &tangent))

	// Positive case
	retrieved, err := DB(ctx).GetTangent(ctx, tangent.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, tangent.ID, retrieved.ID)
	assert.Equal(t, tangent.Status, retrieved.Status)
	assert.Equal(t, tangent.PublicKey, retrieved.PublicKey)

	// Compare JSON data by unmarshaling
	var originalInfo, retrievedInfo map[string]interface{}
	assert.NoError(t, json.Unmarshal(tangent.Info, &originalInfo))
	assert.NoError(t, json.Unmarshal(retrieved.Info, &retrievedInfo))
	assert.Equal(t, originalInfo, retrievedInfo)

	// Negative case - non-existent ID
	_, err = DB(ctx).GetTangent(ctx, uuid.New())
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestUpdateTangent(t *testing.T) {
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

	tangent := models.Tangent{
		ID:        uuid.New(),
		Info:      info.Bytes,
		PublicKey: []byte("test-public-key-update"),
		Status:    "active",
	}
	assert.NoError(t, DB(ctx).CreateTangent(ctx, &tangent))

	// Update status and public key
	tangent.Status = "completed"
	tangent.PublicKey = []byte("test-public-key-updated")
	err := DB(ctx).UpdateTangent(ctx, &tangent)
	assert.NoError(t, err)

	retrieved, err := DB(ctx).GetTangent(ctx, tangent.ID)
	assert.NoError(t, err)
	assert.Equal(t, "completed", retrieved.Status)
	assert.Equal(t, []byte("test-public-key-updated"), retrieved.PublicKey)

	// Update info
	var newInfo pgtype.JSONB
	assert.NoError(t, newInfo.Set(`{"meta": "updated_info"}`))
	tangent.Info = newInfo.Bytes
	err = DB(ctx).UpdateTangent(ctx, &tangent)
	assert.NoError(t, err)

	retrieved, err = DB(ctx).GetTangent(ctx, tangent.ID)
	assert.NoError(t, err)

	// Compare JSON data by unmarshaling
	var originalInfo, retrievedInfo map[string]interface{}
	assert.NoError(t, json.Unmarshal(tangent.Info, &originalInfo))
	assert.NoError(t, json.Unmarshal(retrieved.Info, &retrievedInfo))
	assert.Equal(t, originalInfo, retrievedInfo)

	// Update non-existent tangent
	nonExistentTangent := models.Tangent{
		ID:     uuid.New(),
		Info:   info.Bytes,
		Status: "active",
	}
	err = DB(ctx).UpdateTangent(ctx, &nonExistentTangent)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestDeleteTangent(t *testing.T) {
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

	tangent := models.Tangent{
		ID:        uuid.New(),
		Info:      info.Bytes,
		PublicKey: []byte("test-public-key-delete"),
		Status:    "active",
	}
	assert.NoError(t, DB(ctx).CreateTangent(ctx, &tangent))

	// Delete
	err := DB(ctx).DeleteTangent(ctx, tangent.ID)
	assert.NoError(t, err)

	// Verify
	_, err = DB(ctx).GetTangent(ctx, tangent.ID)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Delete again (should fail as not found)
	err = DB(ctx).DeleteTangent(ctx, tangent.ID)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestListTangents(t *testing.T) {
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

	// Create multiple tangents
	tangents := []models.Tangent{
		{
			ID:        uuid.New(),
			Info:      info.Bytes,
			PublicKey: []byte("test-public-key-1"),
			Status:    "active",
		},
		{
			ID:        uuid.New(),
			Info:      info.Bytes,
			PublicKey: []byte("test-public-key-2"),
			Status:    "pending",
		},
		{
			ID:        uuid.New(),
			Info:      info.Bytes,
			PublicKey: []byte("test-public-key-3"),
			Status:    "completed",
		},
	}

	for i := range tangents {
		assert.NoError(t, DB(ctx).CreateTangent(ctx, &tangents[i]))
	}

	// List tangents
	retrieved, err := DB(ctx).ListTangents(ctx)
	assert.NoError(t, err)
	require.Len(t, retrieved, 3)

	// Verify all tangents are present
	statuses := make(map[string]bool)
	for _, t := range retrieved {
		statuses[t.Status] = true
	}
	assert.True(t, statuses["active"])
	assert.True(t, statuses["pending"])
	assert.True(t, statuses["completed"])
}
