package db

import (
	"context"
	"strings"
	"testing"

	"encoding/json"

	"github.com/jackc/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/uuid"
)

func TestSkillSetOperations(t *testing.T) {
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
	require.NoError(t, err)
	defer DB(ctx).DeleteCatalog(ctx, catalog.CatalogID, "")

	// Create a variant for testing
	variant := models.Variant{
		Name:        "test_variant",
		Description: "A test variant",
		CatalogID:   catalog.CatalogID,
		Info:        info,
	}
	err = DB(ctx).CreateVariant(ctx, &variant)
	assert.NoError(t, err)
	defer DB(ctx).DeleteVariant(ctx, catalog.CatalogID, variant.VariantID, "")

	// Create initial metadata
	initialMetadata := map[string]interface{}{
		"description": "test skillset",
		"skills": []map[string]interface{}{
			{
				"description":     "Test skill",
				"exportedActions": []string{"test.action"},
				"name":            "test-skill",
				"source":          "command-runner",
			},
		},
	}
	initialMetadataBytes, err := json.MarshalIndent(initialMetadata, "", "\t")
	require.NoError(t, err)

	// Create a mock skillset
	ss := &models.SkillSet{
		Path:      "/test/skillset",
		Hash:      "test_hash_123456789012345",
		VariantID: variant.VariantID,
		Metadata:  initialMetadataBytes,
	}

	// Create initial data
	initialData := map[string]interface{}{
		"version": "0.1.0-alpha.1",
		"kind":    "SkillSet",
		"metadata": map[string]interface{}{
			"name":      "test-skillset",
			"catalog":   "test_catalog",
			"namespace": "default",
			"variant":   "test_variant",
		},
		"spec": map[string]interface{}{
			"version": "1.0.0",
			"runners": []map[string]interface{}{
				{
					"name": "command-runner",
					"id":   "system.commandrunner",
					"config": map[string]interface{}{
						"command": "python3 test.py",
					},
				},
			},
			"skills": []map[string]interface{}{
				{
					"description":     "Test skill",
					"exportedActions": []string{"test.action"},
					"name":            "test-skill",
					"source":          "command-runner",
					"inputSchema":     map[string]interface{}{"type": "object"},
					"outputSchema":    map[string]interface{}{"type": "object"},
				},
			},
		},
	}
	initialDataBytes, err := json.MarshalIndent(initialData, "", "\t")
	require.NoError(t, err)

	// Create a mock catalog object
	obj := &models.CatalogObject{
		Hash:     ss.Hash,
		Type:     catcommon.CatalogObjectTypeSkillset,
		Version:  "0.1.0-alpha.1",
		TenantID: tenantID,
		Data:     initialDataBytes,
	}

	// Test UpsertSkillSetObject
	err = DB(ctx).UpsertSkillSetObject(ctx, ss, obj, variant.SkillsetDirectoryID)
	require.NoError(t, err)

	// Test GetSkillSet
	retrievedSS, err := DB(ctx).GetSkillSet(ctx, ss.Path, variant.VariantID, variant.SkillsetDirectoryID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedSS)
	assert.Equal(t, ss.Path, retrievedSS.Path)
	assert.Equal(t, ss.Hash, strings.TrimSpace(retrievedSS.Hash))
	assert.Equal(t, ss.VariantID, retrievedSS.VariantID)

	// Compare metadata as maps
	var expectedMetadata, actualMetadata map[string]interface{}
	err = json.Unmarshal(ss.Metadata, &expectedMetadata)
	assert.NoError(t, err)
	err = json.Unmarshal(retrievedSS.Metadata, &actualMetadata)
	assert.NoError(t, err)
	assert.Equal(t, expectedMetadata, actualMetadata)

	// Test GetSkillSetObject
	retrievedObj, err := DB(ctx).GetSkillSetObject(ctx, ss.Path, variant.SkillsetDirectoryID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedObj)
	assert.Equal(t, obj.Hash, strings.TrimSpace(retrievedObj.Hash))
	assert.Equal(t, obj.Type, retrievedObj.Type)
	assert.Equal(t, obj.Version, retrievedObj.Version)

	// Compare data as maps
	var expectedData, actualData map[string]interface{}
	err = json.Unmarshal(obj.Data, &expectedData)
	assert.NoError(t, err)
	err = json.Unmarshal(retrievedObj.Data, &actualData)
	assert.NoError(t, err)
	assert.Equal(t, expectedData, actualData)

	// Create updated metadata
	updatedMetadata := map[string]interface{}{
		"description": "updated skillset",
		"skills": []map[string]interface{}{
			{
				"description":     "Updated test skill",
				"exportedActions": []string{"test.action", "test.action2"},
				"name":            "test-skill",
				"source":          "command-runner",
			},
		},
	}
	updatedMetadataBytes, err := json.MarshalIndent(updatedMetadata, "", "\t")
	require.NoError(t, err)

	// Test UpdateSkillSet
	ss.Hash = "updated_hash_456789012345678"
	ss.Metadata = updatedMetadataBytes
	err = DB(ctx).UpdateSkillSet(ctx, ss, variant.SkillsetDirectoryID)
	assert.NoError(t, err)

	// Verify update
	updatedSS, err := DB(ctx).GetSkillSet(ctx, ss.Path, variant.VariantID, variant.SkillsetDirectoryID)
	assert.NoError(t, err)
	assert.NotNil(t, updatedSS)
	assert.Equal(t, ss.Hash, strings.TrimSpace(updatedSS.Hash))

	// Compare updated metadata as maps
	var expectedUpdatedMetadata, actualUpdatedMetadata map[string]interface{}
	err = json.Unmarshal(ss.Metadata, &expectedUpdatedMetadata)
	assert.NoError(t, err)
	err = json.Unmarshal(updatedSS.Metadata, &actualUpdatedMetadata)
	assert.NoError(t, err)
	assert.Equal(t, expectedUpdatedMetadata, actualUpdatedMetadata)

	// Test ListSkillSets
	skillsets, err := DB(ctx).ListSkillSets(ctx, variant.SkillsetDirectoryID)
	assert.NoError(t, err)
	assert.NotNil(t, skillsets)
	assert.Len(t, skillsets, 1)
	assert.Equal(t, ss.Path, skillsets[0].Path)
	assert.Equal(t, ss.Hash, strings.TrimSpace(skillsets[0].Hash))

	// Compare listed skillset metadata as maps
	var listedMetadata map[string]interface{}
	err = json.Unmarshal(skillsets[0].Metadata, &listedMetadata)
	assert.NoError(t, err)
	assert.Equal(t, expectedUpdatedMetadata, listedMetadata)

	// Test DeleteSkillSet
	deletedHash, err := DB(ctx).DeleteSkillSet(ctx, ss.Path, variant.SkillsetDirectoryID)
	assert.NoError(t, err)
	assert.Equal(t, ss.Hash, deletedHash)

	// Verify deletion
	deletedSS, err := DB(ctx).GetSkillSet(ctx, ss.Path, variant.VariantID, variant.SkillsetDirectoryID)
	assert.Error(t, err)
	assert.Nil(t, deletedSS)

	// Test error cases
	_, err = DB(ctx).GetSkillSet(ctx, "/nonexistent", variant.VariantID, variant.SkillsetDirectoryID)
	assert.Error(t, err)

	_, err = DB(ctx).GetSkillSetObject(ctx, "/nonexistent", variant.SkillsetDirectoryID)
	assert.Error(t, err)

	// Test with invalid directory ID
	_, err = DB(ctx).GetSkillSet(ctx, ss.Path, variant.VariantID, uuid.Nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrInvalidInput)

	// Test with missing tenant ID
	ctxWithoutTenant := catcommon.WithTenantID(ctx, "")
	_, err = DB(ctx).GetSkillSet(ctxWithoutTenant, ss.Path, variant.VariantID, variant.SkillsetDirectoryID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dberror.ErrMissingTenantID)
}
