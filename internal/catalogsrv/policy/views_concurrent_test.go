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

func TestViewOperations(t *testing.T) {
	//	t.Skip("Skipping concurrent test")
	ctx := newDb()
	defer func() {
		if db.DB(ctx) != nil {
			db.DB(ctx).Close(ctx)
		}
	}()

	tenantID := catcommon.TenantId("TABCDE")
	projectID := catcommon.ProjectId("P12346")
	ctx = catcommon.WithTenantID(ctx, tenantID)
	ctx = catcommon.WithProjectID(ctx, projectID)

	require.NoError(t, db.DB(ctx).CreateTenant(ctx, tenantID))
	defer func() {
		if err := db.DB(ctx).DeleteTenant(ctx, tenantID); err != nil {
			t.Logf("Warning: failed to delete tenant: %v", err)
		}
	}()

	require.NoError(t, db.DB(ctx).CreateProject(ctx, projectID))
	defer func() {
		if err := db.DB(ctx).DeleteProject(ctx, projectID); err != nil {
			t.Logf("Warning: failed to delete project: %v", err)
		}
	}()

	// Create a catalog for testing
	catalogID := uuid.New()
	err := db.DB(ctx).CreateCatalog(ctx, &models.Catalog{
		CatalogID:   catalogID,
		Name:        "test-catalog",
		Description: "Test catalog",
		ProjectID:   projectID,
		Info:        pgtype.JSONB{Status: pgtype.Null},
	})
	require.NoError(t, err)
	defer func() {
		if err := db.DB(ctx).DeleteCatalog(ctx, uuid.Nil, "test-catalog"); err != nil {
			t.Logf("Warning: failed to delete catalog: %v", err)
		}
	}()

	metadata := &interfaces.Metadata{
		Catalog: "test-catalog",
	}

	// Create a WaitGroup to wait for all test cases
	var wg sync.WaitGroup
	// Create a channel to collect errors from all test cases
	errorChan := make(chan error, 100)

	// Helper function to create a new database connection with proper cleanup
	newTestConn := func() (context.Context, func()) {
		ctx := newDb()
		ctx = catcommon.WithTenantID(ctx, tenantID)
		ctx = catcommon.WithProjectID(ctx, projectID)
		return ctx, func() {
			if db.DB(ctx) != nil {
				db.DB(ctx).Close(ctx)
			}
		}
	}

	// Run all test cases concurrently
	testCases := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"concurrent create operations", func(t *testing.T) {
			const numRequests = 5
			var wg sync.WaitGroup
			successChan := make(chan bool, numRequests)
			errorChan := make(chan error, numRequests)

			for i := 0; i < numRequests; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					// Each request gets its own context and connection
					ctx, cleanup := newTestConn()
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

			// Count successes and log errors
			successCount := 0
			for success := range successChan {
				if success {
					successCount++
				}
			}
			for err := range errorChan {
				t.Errorf("Error during concurrent create: %v", err)
			}

			// Verify at least some operations succeeded
			assert.Greater(t, successCount, 0, "At least one create operation should succeed")
		}},
		{"concurrent update operations", func(t *testing.T) {
			// First create a view to update
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
			ctx, cleanup := newTestConn()
			defer cleanup()

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
					// Each request gets its own context and connection
					ctx, cleanup := newTestConn()
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

			// Count successes and log errors
			successCount := 0
			for success := range successChan {
				if success {
					successCount++
				}
			}
			for err := range errorChan {
				t.Errorf("Error during concurrent update: %v", err)
			}

			// Verify at least some operations succeeded
			assert.Greater(t, successCount, 0, "At least one update operation should succeed")

			// Verify the final state
			view, err := db.DB(ctx).GetViewByLabel(ctx, viewName, catalogID)
			require.NoError(t, err)
			assert.Contains(t, view.Description, "Updated description")
		}},
		{"concurrent read operations", func(t *testing.T) {
			// Create a view to read
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

			ctx, cleanup := newTestConn()
			defer cleanup()

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
					// Each request gets its own context and connection
					ctx, cleanup := newTestConn()
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

					// Parse the result into a viewSchema
					var view viewSchema
					if err := json.Unmarshal(result, &view); err != nil {
						errorChan <- fmt.Errorf("read operation %d failed to parse view: %w", i, err)
						return
					}

					firstResultMutex.Lock()
					if firstResult == nil {
						firstResult = &view
					} else {
						// Compare the structured data
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

			// Check for errors
			for err := range errorChan {
				t.Errorf("Error during concurrent read: %v", err)
			}
		}},
		{"concurrent create and read operations", func(t *testing.T) {
			const numRequests = 5
			var wg sync.WaitGroup
			errorChan := make(chan error, numRequests*2) // *2 because each request does both create and read

			for i := 0; i < numRequests; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					// Each request gets its own context and connection
					ctx, cleanup := newTestConn()
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

					// Read the created view
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

			// Check for errors
			for err := range errorChan {
				t.Errorf("Error during concurrent create-read: %v", err)
			}
		}},
		{"concurrent update and read operations", func(t *testing.T) {
			// Create a view to update and read
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

			ctx, cleanup := newTestConn()
			defer cleanup()

			_, err := CreateView(ctx, []byte(initialView), metadata)
			require.NoError(t, err)

			const numRequests = 5
			var wg sync.WaitGroup
			errorChan := make(chan error, numRequests*2) // *2 because each request does both update and read

			for i := 0; i < numRequests; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					// Each request gets its own context and connection
					ctx, cleanup := newTestConn()
					defer cleanup()

					// Update the view
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

					// Read the view
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

			// Check for errors
			for err := range errorChan {
				t.Errorf("Error during concurrent update-read: %v", err)
			}

			// Verify the final state
			view, err := db.DB(ctx).GetViewByLabel(ctx, viewName, catalogID)
			require.NoError(t, err)
			assert.Contains(t, view.Description, "Updated description")
		}},
	}

	// Run all test cases concurrently
	for _, tc := range testCases {
		wg.Add(1)
		go func(tc struct {
			name string
			fn   func(*testing.T)
		}) {
			defer wg.Done()
			// Create a new testing.T for each test case
			tt := &testing.T{}
			tc.fn(tt)
			if tt.Failed() {
				errorChan <- fmt.Errorf("test case %s failed", tc.name)
			}
		}(tc)
	}

	// Wait for all test cases to complete
	wg.Wait()
	close(errorChan)

	// Check for errors from all test cases
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		t.Errorf("Test failed with %d errors:", len(errors))
		for _, err := range errors {
			t.Errorf("  %v", err)
		}
		t.FailNow() // Fail at the very end after all test cases complete
	}
}
