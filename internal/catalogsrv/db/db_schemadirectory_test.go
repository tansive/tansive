package db

import (
	"context"
	"testing"

	"github.com/jackc/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestSchemaDirectory(t *testing.T) {
	// Your test code here
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

	var info pgtype.JSONB
	err = info.Set(`{"key": "value"}`)
	assert.NoError(t, err)

	// Create the catalog for testing
	catalog := models.Catalog{
		Name:        "test_catalog",
		Description: "A test catalog",
		Info:        info,
	}
	err = DB(ctx).CreateCatalog(ctx, &catalog)
	assert.NoError(t, err)
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	variant, err := DB(ctx).GetVariant(ctx, catalog.CatalogID, uuid.Nil, catcommon.DefaultVariant)
	assert.NoError(t, err)

	// get the parameter directory
	rg := variant.ResourceDirectoryID

	dirJson := `
	{
		"/z/a/b/c": {"hash": "a4f7d5b6e8c3d2a9f6e4b3c7d9a5f8b2e6c9d3f5a7e4b8c6d3a9f5e7d2f8b4a6"},
		"/a/b/c2/e/f": {"hash": "7b774effe4a349c6dd82ad4f4f21d34c22b8c323a4df9e20d3d4c61daceca69c"},
		"/a1/b/c/d": {"hash": "e3d4b5a7c6f5e4b3a8f7d2c3b4f6a5d8c7e3b6d9a4c8f3d5a2b6f7e4d3c5a8e9"},
		"/a1/b2/c": {"hash": "d3e6f5b4a9c8f2d7b3e5a6f4c8b7a9e3f5d2b6c4a8f7e9d3a5c6b8e7d2f4c5a9"},
		"/x/y1": {"hash": "8ad13a24fce736b8364d6574b4f9d4a8d2e4f8e0a8d4e5c7c7d6b4a8c3d2a5b6"},
		"/z/a/b2/c": {"hash": "d8f5b3a7e9c6d4f2b8a3f7c5e4d9a6b2f3e7d5c8a9f6b4d3a8f5e2c7d9f4b6a5"},
		"/x": {"hash": "d2d2d2d2b4c4c4a8a8b3b3c3c3e5e5e5f8f8f8e4d4d4b3c3b3d3c3e3f4f5f6f7"},
		"/a/b3/c": {"hash": "c7a4f3b5d9e2f8b6a7d5c3e9a8f4d2b5c6f3a7e8b9d6a3f5c2e7b4d9f8a6c3d2"},
		"/a1/b2": {"hash": "f4d5b8c6a3e9d7f2b6a8e5c3d7f9a6b4e2d3c5a8f7b9d4e6c2f3a7b5e9d8c6f4"},
		"/x1/y/z/a": {"hash": "3f6a5e7d8b9c4f3a2d5b8e7c9d6f4a5b7e2d8a3c9b6f7e4d3a6f8b5c2e9a7d4b"},
		"/m/n": {"hash": "d5d1a7b5e7f7d6a8c5c3d3d8e8b7c6c7d5b4f5e4d6f5b6d4a7f6b3c5d4f4a3e7"},
		"/x1/y2/z3": {"hash": "e5c6f3a7b8d9e4c2a5f7d3b6a8e9f4d2b7c8a3f9d6e2b5c7f4a9e6d3b8f5a2c4"},
		"/x/y2/z/a": {"hash": "7a7a7a7a3e3e3e3e2d2d2d2d3b3b3b3b5a5a5a5a2d2d2d2d1a1a1a1a4c4c4c4c"},
		"/a/b": {"hash": "d1cf9895f816740b576ce1f3e02e9cf4a6b743f252c7ff76bb3e80d59a51d3fc"},
		"/m/n/o/q": {"hash": "d3b5a6c7e8f4d5c3b6a8e7d3f6c4b5a6f7d2e5b3a7f4e3c5d8f6b4d7a5c7f3e9"},
		"/m/n5/o6": {"hash": "e4a9d7f6c3b8e5a2d6c9f3a8d7b5e6c4f2d3b9e8a5c6d7f4a3b8e9c5f6d2a7b4"},
		"/m2/n3/o4/p5": {"hash": "d5b8f6a7c3e4d9f2a6b7e8c5d2a3f9c4b5e7a9d6f3b8a5c7e2d4f9b6c5a3d8e7"},
		"/m/n/o": {"hash": "8e73a4f8c7d8b3f4e3c5d7f5a4c6b4d7e7a4d5b3c5e6d7a5d6f5d4b3c6f7a8d2"},
		"/x1/y/z": {"hash": "b7d5e4a8c6f3d9b6e7c2a4f5d8b3e6c7a5d4f2b9e6a8d3c5f4a2d7b9e3a5c6f7"},
		"/y/z/a": {"hash": "f9a8d4c5e6b3f7a2e4d5b8c7a6f9e3c2d8f5a4b7e6c9d3a5f8b4c2e7d6a3f9b7"},
		"/y/z/a/b/c": {"hash": "c4a5f8d6b9e3d7c2a6f4e5b3a8f7d9c6b5e2a3f9d4e6c8a7f5b2d3f4c6a9e5b8"},
		"/a/b/c2": {"hash": "2e7d2c03a9507ae265ecf5b5356885a53393a5e9fa1e6f8c7d2938a869b7f59b"},
		"/a/b/c2/d": {"hash": "9f9d51bc70ef21ca5c14f307980a29d8c63f73a7f83cb90ff128d3043730d09b"},
		"/a/b3/c/d/e/f": {"hash": "e3f7d5a8c4b6f9e2a3c5d7b8f6a4d3e7c9a5f8b3d6c2f4e9a7d5b3f6c8a2d4e5"},
		"/z/a": {"hash": "e5d8a9f7c6b3a2f4d5e3c8b7d9f6a4c2b5d7e8f3a6c9d4e7b3a8f2d5c9e6b4a7"},
		"/a": {"hash": "3a7bd3e2360a4feecb74b8c9479b5e78d16fdabc2f5e8c94f2e7df8af1a23450"},
		"/a/b/c2/e": {"hash": "73feffa4b7f6bb68e44cf984c85f6e88e6b77a66b95d59c3c2b36633309e9a7d"},
		"/x2/y/z": {"hash": "f7a5d4c6b3e8f9a2c5e7b4d6f3a9e5c7d2f8b5a4c9d3e6f7b8a9c3f4d2e5b7a6"},
		"/a/b/c1": {"hash": "5d41402abc4b2a76b9719d911017c592c63f73a7f83cb90ff128d3043730d09b"},
		"/a1": {"hash": "3b6f4a3c2d1a8c5f7e6d8b3a6c3d5a8e7f4d3b2a7f6e8d9c4b5f6d3a8c7e5b4"},
		"/x/y2/z": {"hash": "f4a7b1b4c3e5e5b4c4d4d4f8f4e3c3a4b3c4e5d4f5b5a4c5f6e5d7b8a7c6d9b6"},
		"/x/y2": {"hash": "d47b127bc2f4c3e5b4e3a4f8f4e3c4a8b5d4e6a5b6c3a4d4e4b5a5e4b6c3d2d2"},
		"/z/a/b": {"hash": "d3b7a4c5f6e2d8b3f9a5c7e4f8d2b6a9e5c3f4a7d6b8e3f5a9c6b2d7e9f4c8a3"},
		"/x1/y/z/a/b": {"hash": "a2d4c6b3f7e5a4d9b6c5a8e7f2d3c4b8f6d7a9e5b2c6a3f4d8b7e5c9f2d3a6e4"},
		"/m/n/o/p": {"hash": "c3f3d2b4e5c3a7f6d7a5b4c6f8d4e3b5d7e7a8c3d4f5e6a4b4f6d5c3a5b3e7c6"},
		"/m/n/o/p/q": {"hash": "a4b5c3d2f4a6e5d3c7f5d4a4f3b6e7d3f6c5a7f8d5e3a7b6e4a6f8d2e3c4d7a5"},
		"/m/n2": {"hash": "b4e7f6a5d3c8f2b9e4d7a3f8c6b5a7e3d9c5f4b2a8e7c6d5a9f3b7e4c2d6a5f9"},
		"/y/z/a/b": {"hash": "d3f6b7a4e9c5d8f2b4a6e5c7d3a8f9b6c3e7d5a9f4b2c8e6d7a5b4f3c9a6d2e8"},
		"/a1/b/c": {"hash": "8a7d6b4c3e2f1a9d7c5b3e4f6a8b9d3e7c5f4a2d8b3c6f7a4e5d6c8f3b7a9d5"},
		"/m/n/o/p/q/r": {"hash": "b5d8f7a3c9e6d2b7a5f4e8c6b3a9d5f7c4e2b6f9a3c5d7e8b2f4a9c3f5d8a7e6"},
		"/a1/b": {"hash": "d4a5f8c7b6e4d3a5c4f7b8a6e5d2c7f3b5a4c3e6d8f9b4a7d6c8e5b4f7a5c6d8"},
		"/a1/b/c/d/e": {"hash": "c2b5d3f7e9a8d6f4b7a3c5e6f9d2b4a7f3e5c8d4f6a9b7e3d8c6f5b2a7d9c4e6"},
		"/a1/b/c/e/f": {"hash": "7b774effe4a349c6dd82ad4f4f21d34c22b8c323a4df9e20d3d4c61daceca69c"},
		"/x1/y/z/a/b/c": {"hash": "d6c7e8b4a9f3b2e7c4f5d3a6e8c9b5f4d7e6a3b9d2f5c8e7b6f3d5a9e8c2a4d7"},
		"/a2/b3": {"hash": "c6e7f3a9d8b5f4a7e2c9b3d6f8a5c4b7e6d3a2f9b5d8e3a7f4c5d6a3e8b7f9c4"},
		"/x/y2/z/a/b": {"hash": "b7f6e5c3d4f8b2a9c6e7f4a3d5b8e6f7a2c4d9e3a8c5f6d2b7e9c4a6f8b3d5e7"},
		"/x1/y2": {"hash": "e4a8b9d3f7c5d2a6e3f8b5c7d4f9a2e6d7c8b3a5f6e7c9a4d5f2b8e3a6c7d9f4"},
		"/m/n2/o": {"hash": "c8f3d6b5e4a7f9b3e6d8c2f5a9d4b7c3a5f4d9e8b6a3c5f7e9d2a8f6c4b7e5d9"},
		"/a/b2": {"hash": "e7d6c5b4a3f2b9d8e6a5c7f8b3d4a2e5d7c6b8f9a4e3c7d5b2a6f4c9e8a3d7f6"},
		"/a1/b3": {"hash": "c7d5e4a9b6f3e8c2d7a5b4f6c9a8e3d5f7a4c6b2e9d3a5f8b7c4e6a2d3f9b5c8"},
		"/z/a/b/c/d/e/f": {"hash": "c2b5d3f7e9a8d6f4b7a3c5e6f9d2b4a7f3e5c8d4f6a9b7e3d8c6f5b2a7d9c4e6"},
		"/p/q/r/s": {"hash": "d5e6f7a8b9c4d3e2f1a6b5c4d3e2f1a6b5c4d3e2f1a6b5c4d3e2f1a6b5c4d3e2"},
		"/m/n/s": {"hash": "d5e6f7a8b9c4d3e2f1a6b5c4d3e2f1a6b5c4d3e2f1a6b5c4d3e2f1a6b5c4d3e2"},
		"/m/n/p/s": {"hash": "d5e6f7a8b9c4d3e2f1a6b5c4d3e2f1a6b5c4d3e2f1a6b5c4d3e2f1a6b5c4d3e2"}
	}
	`
	dir, err := models.JSONToDirectory([]byte(dirJson))
	assert.NoError(t, err)
	err = DB(ctx).SetDirectory(ctx, catcommon.CatalogObjectTypeResource, rg, []byte(dirJson))
	assert.NoError(t, err)
	// get the directory
	dirRetJson, err := DB(ctx).GetDirectory(ctx, catcommon.CatalogObjectTypeResource, rg)
	assert.NoError(t, err)
	dirRet, err := models.JSONToDirectory(dirRetJson)
	assert.NoError(t, err)
	assert.Equal(t, dir, dirRet)
	assert.Equal(t, dirRet["/a2/b3"], dir["/a2/b3"])

	// Test the GetObjectByPath function
	object, err := DB(ctx).GetObjectRefByPath(ctx, catcommon.CatalogObjectTypeResource, rg, "/a/b3/c/d/e/f")
	assert.NoError(t, err)
	assert.Equal(t, object.Hash, dir["/a/b3/c/d/e/f"].Hash)

	// Non existing path
	object, err = DB(ctx).GetObjectRefByPath(ctx, catcommon.CatalogObjectTypeResource, rg, "/non/existing/path")
	assert.ErrorIs(t, err, dberror.ErrNotFound)
	assert.Nil(t, object)

	// Does path exist
	exists, err := DB(ctx).PathExists(ctx, catcommon.CatalogObjectTypeResource, rg, "/x/y2/z/a/b")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Non existing path
	exists, err = DB(ctx).PathExists(ctx, catcommon.CatalogObjectTypeResource, rg, "/non/existing/path")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Update object by path
	updateObj := models.ObjectRef{
		Hash: "new_hash_value",
	}
	err = DB(ctx).AddOrUpdateObjectByPath(ctx, catcommon.CatalogObjectTypeResource, rg, "/a/b3/c/d/e/f", updateObj)
	assert.NoError(t, err)
	object, err = DB(ctx).GetObjectRefByPath(ctx, catcommon.CatalogObjectTypeResource, rg, "/a/b3/c/d/e/f")
	assert.NoError(t, err)
	assert.Equal(t, object.Hash, updateObj.Hash)

	// Update non existing path
	err = DB(ctx).AddOrUpdateObjectByPath(ctx, catcommon.CatalogObjectTypeResource, rg, "/non/existing/path", updateObj)
	// this should have added a new entry
	assert.NoError(t, err)
	object, err = DB(ctx).GetObjectRefByPath(ctx, catcommon.CatalogObjectTypeResource, rg, "/non/existing/path")
	assert.NoError(t, err)
	assert.Equal(t, object.Hash, updateObj.Hash)

	// Delete object by path
	hash, err := DB(ctx).DeleteObjectByPath(ctx, catcommon.CatalogObjectTypeResource, rg, "/a/b3/c/d/e/f")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	object, err = DB(ctx).GetObjectRefByPath(ctx, catcommon.CatalogObjectTypeResource, rg, "/a/b3/c/d/e/f")
	assert.ErrorIs(t, err, dberror.ErrNotFound)
	assert.Nil(t, object)
}
