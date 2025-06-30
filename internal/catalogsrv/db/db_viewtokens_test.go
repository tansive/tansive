package db

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestCreateViewToken(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create test tenant
	err := DB(ctx).CreateTenant(ctx, tenantID)
	require.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	require.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	viewID := uuid.New()
	token := &models.ViewToken{
		ViewID:   viewID,
		ExpireAt: time.Now().Add(time.Hour),
	}

	// Test successful creation
	err = DB(ctx).CreateViewToken(ctx, token)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, token.TokenID)
	assert.NotZero(t, token.CreatedAt)
	assert.NotZero(t, token.UpdatedAt)

	// Test creation with nil view ID
	invalidToken := &models.ViewToken{
		ExpireAt: time.Now().Add(time.Hour),
	}
	err = DB(ctx).CreateViewToken(ctx, invalidToken)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)
}

func TestGetViewToken(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create test tenant
	err := DB(ctx).CreateTenant(ctx, tenantID)
	require.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	require.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Create a token first
	viewID := uuid.New()
	token := &models.ViewToken{
		ViewID:   viewID,
		ExpireAt: time.Now().Add(time.Hour),
	}
	err = DB(ctx).CreateViewToken(ctx, token)
	require.NoError(t, err)

	// Test successful retrieval
	retrieved, err := DB(ctx).GetViewToken(ctx, token.TokenID)
	assert.NoError(t, err)
	assert.Equal(t, token.TokenID, retrieved.TokenID)
	assert.Equal(t, token.ViewID, retrieved.ViewID)
	assert.Equal(t, token.ExpireAt.Unix(), retrieved.ExpireAt.Unix())

	// Test retrieval of non-existent token
	_, err = DB(ctx).GetViewToken(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestUpdateViewTokenExpiry(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create test tenant
	err := DB(ctx).CreateTenant(ctx, tenantID)
	require.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	require.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Create a token first
	viewID := uuid.New()
	token := &models.ViewToken{
		ViewID:   viewID,
		ExpireAt: time.Now().Add(time.Hour),
	}
	err = DB(ctx).CreateViewToken(ctx, token)
	require.NoError(t, err)

	// Test successful update
	newExpiry := time.Now().Add(2 * time.Hour)
	err = DB(ctx).UpdateViewTokenExpiry(ctx, token.TokenID, newExpiry)
	assert.NoError(t, err)

	// Verify the update
	updated, err := DB(ctx).GetViewToken(ctx, token.TokenID)
	assert.NoError(t, err)
	assert.Equal(t, newExpiry.Unix(), updated.ExpireAt.Unix())

	// Test update of non-existent token
	err = DB(ctx).UpdateViewTokenExpiry(ctx, uuid.New(), newExpiry)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}

func TestDeleteViewToken(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	ctx = newDb(ctx)
	defer DB(ctx).Close(ctx)

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12345")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	// Create test tenant
	err := DB(ctx).CreateTenant(ctx, tenantID)
	require.NoError(t, err)
	defer DB(ctx).DeleteTenant(ctx, tenantID)

	err = DB(ctx).CreateProject(ctx, projectID)
	require.NoError(t, err)
	defer DB(ctx).DeleteProject(ctx, projectID)

	// Create a token first
	viewID := uuid.New()
	token := &models.ViewToken{
		ViewID:   viewID,
		ExpireAt: time.Now().Add(time.Hour),
	}
	err = DB(ctx).CreateViewToken(ctx, token)
	require.NoError(t, err)

	// Test successful deletion
	err = DB(ctx).DeleteViewToken(ctx, token.TokenID)
	assert.NoError(t, err)

	// Verify the deletion
	_, err = DB(ctx).GetViewToken(ctx, token.TokenID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)

	// Test deletion of non-existent token
	err = DB(ctx).DeleteViewToken(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrNotFound)
}
