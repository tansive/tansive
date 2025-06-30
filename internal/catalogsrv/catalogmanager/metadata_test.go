package catalogmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/pkg/types"
)

func TestGetMetadata(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		expectedError bool
		expectedMeta  *interfaces.Metadata
	}{
		{
			name: "valid metadata",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "ResourceGroup",
				"metadata": {
					"name": "test-group",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default"
				}
			}`,
			expectedError: false,
			expectedMeta: &interfaces.Metadata{
				Name:      "test-group",
				Catalog:   "test-catalog",
				Namespace: types.NullableStringFrom("default"),
				Variant:   types.NullableStringFrom("default"),
			},
		},
		{
			name:          "empty input",
			jsonInput:     "",
			expectedError: true,
			expectedMeta:  nil,
		},
		{
			name: "invalid JSON",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "ResourceGroup",
				"metadata": {
					"name": "test-group",
					"catalog": "test-catalog",
					"namespace": "default",
					"variant": "default"
				}
			`, // missing closing brace
			expectedError: true,
			expectedMeta:  nil,
		},
		{
			name: "missing metadata",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "ResourceGroup"
			}`,
			expectedError: true,
			expectedMeta:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := getMetadata(context.Background(), []byte(tt.jsonInput))
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, meta)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMeta, meta)
			}
		})
	}
}

func TestCanonicalizeMetadata(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		kind          string
		metadata      *interfaces.Metadata
		expectedError bool
		expectedMeta  *interfaces.Metadata
		checkJSON     func(t *testing.T, json []byte)
	}{
		{
			name: "update existing metadata",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "ResourceGroup",
				"metadata": {
					"name": "old-name",
					"catalog": "old-catalog",
					"namespace": "old-namespace",
					"variant": "old-variant"
				}
			}`,
			kind: "ResourceGroup",
			metadata: &interfaces.Metadata{
				Name:      "new-name",
				Catalog:   "new-catalog",
				Namespace: types.NullableStringFrom("new-namespace"),
				Variant:   types.NullableStringFrom("new-variant"),
			},
			expectedError: false,
			expectedMeta: &interfaces.Metadata{
				Name:      "new-name",
				Catalog:   "new-catalog",
				Namespace: types.NullableStringFrom("new-namespace"),
				Variant:   types.NullableStringFrom("new-variant"),
			},
			checkJSON: func(t *testing.T, json []byte) {
				assert.Contains(t, string(json), `"name":"new-name"`)
				assert.Contains(t, string(json), `"catalog":"new-catalog"`)
				assert.Contains(t, string(json), `"namespace":"new-namespace"`)
				assert.Contains(t, string(json), `"variant":"new-variant"`)
			},
		},
		{
			name: "partial update",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "ResourceGroup",
				"metadata": {
					"name": "old-name",
					"catalog": "old-catalog",
					"namespace": "old-namespace",
					"variant": "old-variant"
				}
			}`,
			kind: "ResourceGroup",
			metadata: &interfaces.Metadata{
				Name: "new-name",
			},
			expectedError: false,
			expectedMeta: &interfaces.Metadata{
				Name:      "new-name",
				Catalog:   "old-catalog",
				Namespace: types.NullableStringFrom("old-namespace"),
				Variant:   types.NullableStringFrom("old-variant"),
			},
			checkJSON: func(t *testing.T, json []byte) {
				assert.Contains(t, string(json), `"name":"new-name"`)
				assert.Contains(t, string(json), `"catalog":"old-catalog"`)
			},
		},
		{
			name: "set default variant",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "ResourceGroup",
				"metadata": {
					"name": "test-group",
					"catalog": "test-catalog"
				}
			}`,
			kind:          "ResourceGroup",
			metadata:      nil,
			expectedError: false,
			expectedMeta: &interfaces.Metadata{
				Name:    "test-group",
				Catalog: "test-catalog",
				Variant: types.NullableStringFrom(catcommon.DefaultVariant),
			},
			checkJSON: func(t *testing.T, json []byte) {
				assert.Contains(t, string(json), `"variant":"default"`)
			},
		},
		{
			name:          "empty input",
			jsonInput:     "",
			kind:          "ResourceGroup",
			metadata:      nil,
			expectedError: true,
			expectedMeta:  nil,
		},
		{
			name: "invalid JSON",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "ResourceGroup",
				"metadata": {
					"name": "test-group"
				}
			`, // missing closing brace
			kind:          "ResourceGroup",
			metadata:      nil,
			expectedError: true,
			expectedMeta:  nil,
		},
		{
			name: "missing metadata",
			jsonInput: `{
				"apiVersion": "0.1.0-alpha.1",
				"kind": "ResourceGroup"
			}`,
			kind:          "ResourceGroup",
			metadata:      nil,
			expectedError: true,
			expectedMeta:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			json, meta, err := canonicalizeMetadata([]byte(tt.jsonInput), tt.kind, tt.metadata)
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, meta)
				assert.Nil(t, json)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMeta, meta)
				if tt.checkJSON != nil {
					tt.checkJSON(t, json)
				}
			}
		})
	}
}
