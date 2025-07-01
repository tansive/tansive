package policy

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"encoding/json"

	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/uuid"
)

// --- Common setup helpers ---
func setupTestCatalog(t *testing.T) (ctx context.Context, tenantID catcommon.TenantId, projectID catcommon.ProjectId, catalogID uuid.UUID, cleanup func()) {
	tenantID = catcommon.TenantId("TABCDE")
	projectID = catcommon.ProjectId("P12346")
	ctx = newDb()
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	require.NoError(t, db.DB(ctx).CreateTenant(ctx, tenantID))
	require.NoError(t, db.DB(ctx).CreateProject(ctx, projectID))

	catalogID = uuid.New()
	err := db.DB(ctx).CreateCatalog(ctx, &models.Catalog{
		CatalogID:   catalogID,
		Name:        "test-catalog",
		Description: "Test catalog",
		ProjectID:   projectID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)

	cleanup = func() {
		_ = db.DB(ctx).DeleteCatalog(ctx, uuid.Nil, "test-catalog")
		_ = db.DB(ctx).DeleteProject(ctx, projectID)
		_ = db.DB(ctx).DeleteTenant(ctx, tenantID)
		if db.DB(ctx) != nil {
			db.DB(ctx).Close(ctx)
		}
	}
	return
}

func newTestConnWithIDs(tenantID catcommon.TenantId, projectID catcommon.ProjectId) (context.Context, func()) {
	ctx := newDb()
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)
	return ctx, func() {
		if db.DB(ctx) != nil {
			db.DB(ctx).Close(ctx)
		}
	}
}

// --- Individual test cases ---

func TestViewConcurrentCreate(t *testing.T) {
	_, tenantID, projectID, _, cleanup := setupTestCatalog(t)
	defer t.Cleanup(cleanup)
	metadata := &interfaces.Metadata{Catalog: "test-catalog"}

	const numRequests = 5
	var wg sync.WaitGroup
	successChan := make(chan bool, numRequests)
	errorChan := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, cleanup := newTestConnWithIDs(tenantID, projectID)
			defer cleanup()
			viewJSON := `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "View",
				"metadata": {
					"name": "view-%d",
					"catalog": "test-catalog",
					"description": "Test view %d"
				},
				"spec": {
					"rules": [{
						"intent": "Allow",
						"actions": ["system.catalog.list"],
						"targets": ["res://variants/my-variant"]
					}]
				}
			}`
			viewJSON = fmt.Sprintf(viewJSON, i, i)
			_, err := CreateView(ctx, []byte(viewJSON), metadata)
			if err != nil {
				errorChan <- fmt.Errorf("create operation %d failed: %w", i, err)
				successChan <- false
			} else {
				successChan <- true
			}
		}(i)
	}
	wg.Wait()
	close(successChan)
	close(errorChan)

	successCount := 0
	for success := range successChan {
		if success {
			successCount++
		}
	}
	for err := range errorChan {
		t.Errorf("Error during concurrent create: %v", err)
	}
	assert.Greater(t, successCount, 0, "At least one create operation should succeed")
}

func TestViewConcurrentUpdate(t *testing.T) {
	ctx, tenantID, projectID, catalogID, cleanup := setupTestCatalog(t)
	defer cleanup()
	metadata := &interfaces.Metadata{Catalog: "test-catalog"}
	viewName := "update-view"
	initialView := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "update-view",
			"catalog": "test-catalog",
			"description": "Initial description"
		},
		"spec": {
			"rules": [{
				"intent": "Allow",
				"actions": ["system.catalog.list"],
				"targets": ["res://variants/my-variant"]
			}]
		}
	}`
	_, err := CreateView(ctx, []byte(initialView), metadata)
	require.NoError(t, err)

	const numRequests = 3
	var wg sync.WaitGroup
	successChan := make(chan bool, numRequests)
	errorChan := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, cleanup := newTestConnWithIDs(tenantID, projectID)
			defer cleanup()
			updateView := `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "View",
				"metadata": {
					"name": "update-view",
					"catalog": "test-catalog",
					"description": "Updated description %d"
				},
				"spec": {
					"rules": [{
						"intent": "Allow",
						"actions": ["system.catalog.list", "system.variant.list"],
						"targets": ["res://variants/my-variant"]
					}]
				}
			}`
			updateView = fmt.Sprintf(updateView, i)
			_, err := UpdateView(ctx, []byte(updateView), metadata)
			if err != nil {
				errorChan <- fmt.Errorf("update operation %d failed: %w", i, err)
				successChan <- false
			} else {
				successChan <- true
			}
		}(i)
	}
	wg.Wait()
	close(successChan)
	close(errorChan)

	successCount := 0
	for success := range successChan {
		if success {
			successCount++
		}
	}
	for err := range errorChan {
		t.Errorf("Error during concurrent update: %v", err)
	}
	assert.Greater(t, successCount, 0, "At least one update operation should succeed")
	view, err := db.DB(ctx).GetViewByLabel(ctx, viewName, catalogID)
	require.NoError(t, err)
	assert.Contains(t, view.Description, "Updated description")
}

func TestViewConcurrentRead(t *testing.T) {
	ctx, tenantID, projectID, catalogID, cleanup := setupTestCatalog(t)
	defer cleanup()
	metadata := &interfaces.Metadata{Catalog: "test-catalog"}
	viewName := "read-view"
	viewJSON := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "read-view",
			"catalog": "test-catalog",
			"description": "View for reads"
		},
		"spec": {
			"rules": [{
				"intent": "Allow",
				"actions": ["system.catalog.list"],
				"targets": ["res://variants/my-variant"]
			}]
		}
	}`
	_, err := CreateView(ctx, []byte(viewJSON), metadata)
	require.NoError(t, err)

	const numRequests = 10
	var wg sync.WaitGroup
	errorChan := make(chan error, numRequests)
	var firstResult *viewSchema
	var firstResultMutex sync.Mutex

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, cleanup := newTestConnWithIDs(tenantID, projectID)
			defer cleanup()
			reqCtx := interfaces.RequestContext{
				CatalogID:  catalogID,
				Catalog:    "test-catalog",
				ObjectName: viewName,
			}
			vr, err := NewViewKindHandler(ctx, reqCtx)
			if err != nil {
				errorChan <- fmt.Errorf("read operation %d failed to create handler: %w", i, err)
				return
			}
			result, err := vr.Get(ctx)
			if err != nil {
				errorChan <- fmt.Errorf("read operation %d failed to get view: %w", i, err)
				return
			}
			var view viewSchema
			if err := json.Unmarshal(result, &view); err != nil {
				errorChan <- fmt.Errorf("read operation %d failed to parse view: %w", i, err)
				return
			}
			firstResultMutex.Lock()
			if firstResult == nil {
				firstResult = &view
			} else {
				assert.Equal(t, firstResult.ApiVersion, view.ApiVersion)
				assert.Equal(t, firstResult.Kind, view.Kind)
				assert.Equal(t, firstResult.Metadata.Name, view.Metadata.Name)
				assert.Equal(t, firstResult.Metadata.Catalog, view.Metadata.Catalog)
				assert.Equal(t, firstResult.Metadata.Description, view.Metadata.Description)
				assert.Equal(t, firstResult.Spec.Rules, view.Spec.Rules)
			}
			firstResultMutex.Unlock()
		}(i)
	}
	wg.Wait()
	close(errorChan)
	for err := range errorChan {
		t.Errorf("Error during concurrent read: %v", err)
	}
}

func TestViewConcurrentCreateRead(t *testing.T) {
	_, tenantID, projectID, catalogID, cleanup := setupTestCatalog(t)
	defer cleanup()
	metadata := &interfaces.Metadata{Catalog: "test-catalog"}
	const numRequests = 5
	var wg sync.WaitGroup
	errorChan := make(chan error, numRequests*2)
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, cleanup := newTestConnWithIDs(tenantID, projectID)
			defer cleanup()
			viewJSON := `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "View",
				"metadata": {
					"name": "cr-view-%d",
					"catalog": "test-catalog",
					"description": "Create-read test view %d"
				},
				"spec": {
					"rules": [{
						"intent": "Allow",
						"actions": ["system.catalog.list"],
						"targets": ["res://variants/my-variant"]
					}]
				}
			}`
			viewJSON = fmt.Sprintf(viewJSON, i, i)
			_, err := CreateView(ctx, []byte(viewJSON), metadata)
			if err != nil {
				errorChan <- fmt.Errorf("create-read operation %d failed to create: %w", i, err)
				return
			}
			reqCtx := interfaces.RequestContext{
				CatalogID:  catalogID,
				Catalog:    "test-catalog",
				ObjectName: fmt.Sprintf("cr-view-%d", i),
			}
			vr, err := NewViewKindHandler(ctx, reqCtx)
			if err != nil {
				errorChan <- fmt.Errorf("create-read operation %d failed to create handler: %w", i, err)
				return
			}
			_, err = vr.Get(ctx)
			if err != nil {
				errorChan <- fmt.Errorf("create-read operation %d failed to read: %w", i, err)
			}
		}(i)
	}
	wg.Wait()
	close(errorChan)
	for err := range errorChan {
		t.Errorf("Error during concurrent create-read: %v", err)
	}
}

func TestViewConcurrentUpdateRead(t *testing.T) {
	ctx, tenantID, projectID, catalogID, cleanup := setupTestCatalog(t)
	defer cleanup()
	metadata := &interfaces.Metadata{Catalog: "test-catalog"}
	viewName := "ur-view"
	initialView := `{
		"apiVersion": "0.1.0-alpha.1",
		"kind": "View",
		"metadata": {
			"name": "ur-view",
			"catalog": "test-catalog",
			"description": "Initial description"
		},
		"spec": {
			"rules": [{
				"intent": "Allow",
				"actions": ["system.catalog.list"],
				"targets": ["res://variants/my-variant"]
			}]
		}
	}`
	_, err := CreateView(ctx, []byte(initialView), metadata)
	require.NoError(t, err)
	const numRequests = 5
	var wg sync.WaitGroup
	errorChan := make(chan error, numRequests*2)
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, cleanup := newTestConnWithIDs(tenantID, projectID)
			defer cleanup()
			updateView := `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "View",
				"metadata": {
					"name": "ur-view",
					"catalog": "test-catalog",
					"description": "Updated description %d"
				},
				"spec": {
					"rules": [{
						"intent": "Allow",
						"actions": ["system.catalog.list", "system.variant.list"],
						"targets": ["res://variants/my-variant"]
					}]
				}
			}`
			updateView = fmt.Sprintf(updateView, i)
			_, err := UpdateView(ctx, []byte(updateView), metadata)
			if err != nil {
				errorChan <- fmt.Errorf("update-read operation %d failed to update: %w", i, err)
				return
			}
			reqCtx := interfaces.RequestContext{
				CatalogID:  catalogID,
				Catalog:    "test-catalog",
				ObjectName: viewName,
			}
			vr, err := NewViewKindHandler(ctx, reqCtx)
			if err != nil {
				errorChan <- fmt.Errorf("update-read operation %d failed to create handler: %w", i, err)
				return
			}
			_, err = vr.Get(ctx)
			if err != nil {
				errorChan <- fmt.Errorf("update-read operation %d failed to read: %w", i, err)
			}
		}(i)
	}
	wg.Wait()
	close(errorChan)
	for err := range errorChan {
		t.Errorf("Error during concurrent update-read: %v", err)
	}
	view, err := db.DB(ctx).GetViewByLabel(ctx, viewName, catalogID)
	require.NoError(t, err)
	assert.Contains(t, view.Description, "Updated description")
}
