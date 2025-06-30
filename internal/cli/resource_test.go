package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrictMapAnyAnyToStringAny(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expected    map[string]any
		expectError bool
	}{
		{
			name: "simple map with string keys",
			input: map[any]any{
				"name":  "test",
				"value": 42,
			},
			expected: map[string]any{
				"name":  "test",
				"value": 42,
			},
			expectError: false,
		},
		{
			name: "nested map structure",
			input: map[any]any{
				"metadata": map[any]any{
					"name":   "test-resource",
					"labels": map[any]any{"env": "prod"},
				},
				"spec": map[any]any{
					"version": "v1",
					"config":  map[any]any{"enabled": true},
				},
			},
			expected: map[string]any{
				"metadata": map[string]any{
					"name":   "test-resource",
					"labels": map[string]any{"env": "prod"},
				},
				"spec": map[string]any{
					"version": "v1",
					"config":  map[string]any{"enabled": true},
				},
			},
			expectError: false,
		},
		{
			name: "map with array values",
			input: map[any]any{
				"items": []any{
					map[any]any{"id": 1, "name": "item1"},
					map[any]any{"id": 2, "name": "item2"},
				},
				"count": 2,
			},
			expected: map[string]any{
				"items": []any{
					map[string]any{"id": 1, "name": "item1"},
					map[string]any{"id": 2, "name": "item2"},
				},
				"count": 2,
			},
			expectError: false,
		},
		{
			name:        "empty map",
			input:       map[any]any{},
			expected:    map[string]any{},
			expectError: false,
		},
		{
			name: "map with primitive values",
			input: map[any]any{
				"string": "hello",
				"int":    123,
				"float":  3.14,
				"bool":   true,
				"null":   nil,
			},
			expected: map[string]any{
				"string": "hello",
				"int":    123,
				"float":  3.14,
				"bool":   true,
				"null":   nil,
			},
			expectError: false,
		},
		{
			name: "non-string key error",
			input: map[any]any{
				123: "value",
			},
			expectError: true,
		},
		{
			name:        "nil input",
			input:       nil,
			expectError: true,
		},
		{
			name:        "non-map input",
			input:       "not a map",
			expectError: true,
		},
		{
			name: "nested non-string key error",
			input: map[any]any{
				"valid": map[any]any{
					456: "invalid key",
				},
			},
			expectError: true,
		},
		{
			name: "array with non-string key error",
			input: map[any]any{
				"items": []any{
					map[any]any{789: "invalid key"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StrictMapAnyAnyToStringAny(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestStrictMapAnyAnyToStringAnyJSONRoundTrip(t *testing.T) {
	// Test that the conversion works correctly with JSON round-trip
	input := map[any]any{
		"metadata": map[any]any{
			"name":   "test-resource",
			"labels": map[any]any{"env": "prod", "version": "v1"},
		},
		"spec": map[any]any{
			"config": map[any]any{
				"enabled": true,
				"items":   []any{"item1", "item2", "item3"},
			},
		},
	}

	// Convert using our function
	result, err := StrictMapAnyAnyToStringAny(input)
	assert.NoError(t, err)

	// Convert to JSON and back to verify structure
	jsonData, err := json.Marshal(result)
	assert.NoError(t, err)

	var roundTrip map[string]any
	err = json.Unmarshal(jsonData, &roundTrip)
	assert.NoError(t, err)

	// Verify the round-trip matches our result
	assert.Equal(t, result, roundTrip)
}

func TestLoadResourceFromMultiYAMLFile(t *testing.T) {
	// Test data definitions
	validMultiResourceYAML := []byte(`---
kind: Catalog
metadata:
  name: test-catalog
  labels:
    env: test
    version: v1
spec:
  description: A test catalog
  version: 1.0.0
---
kind: Namespace
metadata:
  name: test-namespace
  labels:
    env: prod
spec:
  description: A test namespace
---
kind: Variant
metadata:
  name: test-variant
  labels:
    type: development
spec:
  catalog: test-catalog
  version: dev-v1
---
kind: View
metadata:
  name: test-view
  labels:
    type: production
spec:
  variant: test-variant
  filters:
    - field: status
      value: active
---
kind: Resource
metadata:
  name: test-resource
  labels:
    category: data
spec:
  type: database
  config:
    host: localhost
    port: 5432
---
kind: SkillSet
metadata:
  name: test-skillset
  labels:
    domain: ml
spec:
  skills:
    - name: python
      level: expert
    - name: tensorflow
      level: intermediate`)

	singleResourceYAML := []byte(`kind: Catalog
metadata:
  name: single-catalog
  labels:
    env: single
spec:
  description: A single catalog resource
  version: 2.0.0`)

	emptyYAML := []byte(``)

	invalidKindYAML := []byte(`---
kind: InvalidKind
metadata:
  name: invalid-resource
spec:
  description: This should fail
---
kind: Catalog
metadata:
  name: valid-catalog
spec:
  description: This should work`)

	missingMetadataYAML := []byte(`---
kind: Catalog
spec:
  description: Missing metadata
---
kind: Namespace
metadata:
  name: valid-namespace
spec:
  description: This should work`)

	invalidMetadataYAML := []byte(`---
kind: Catalog
metadata: "not a map"
spec:
  description: Invalid metadata format
---
kind: Namespace
metadata:
  name: valid-namespace
spec:
  description: This should work`)

	multipleSameTypeYAML := []byte(`---
kind: Catalog
metadata:
  name: catalog1
spec:
  description: First catalog
---
kind: Catalog
metadata:
  name: catalog2
spec:
  description: Second catalog
---
kind: Namespace
metadata:
  name: ns1
spec:
  description: First namespace
---
kind: Namespace
metadata:
  name: ns2
spec:
  description: Second namespace
`)

	tests := []struct {
		name           string
		filename       string
		yamlData       []byte
		expectError    bool
		expectedKinds  []string
		expectedCounts map[string]int
	}{
		{
			name:        "valid multi-resource file",
			filename:    "dummy.yaml",
			yamlData:    validMultiResourceYAML,
			expectError: false,
			expectedKinds: []string{
				KindCatalog, KindNamespace, KindVariant, KindView, KindResource, KindSkillset,
			},
			expectedCounts: map[string]int{
				KindCatalog:   1,
				KindNamespace: 1,
				KindVariant:   1,
				KindView:      1,
				KindResource:  1,
				KindSkillset:  1,
			},
		},
		{
			name:        "single resource file",
			filename:    "dummy.yaml",
			yamlData:    singleResourceYAML,
			expectError: false,
			expectedKinds: []string{
				KindCatalog,
			},
			expectedCounts: map[string]int{
				KindCatalog: 1,
			},
		},
		{
			name:           "empty file",
			filename:       "dummy.yaml",
			yamlData:       emptyYAML,
			expectError:    false,
			expectedKinds:  []string{},
			expectedCounts: map[string]int{},
		},
		{
			name:        "invalid resource kind",
			filename:    "dummy.yaml",
			yamlData:    invalidKindYAML,
			expectError: true,
		},
		{
			name:        "missing metadata",
			filename:    "dummy.yaml",
			yamlData:    missingMetadataYAML,
			expectError: true,
		},
		{
			name:        "invalid metadata format",
			filename:    "dummy.yaml",
			yamlData:    invalidMetadataYAML,
			expectError: true,
		},
		{
			name:        "multiple resources of same type",
			filename:    "dummy.yaml",
			yamlData:    multipleSameTypeYAML,
			expectError: false,
			expectedKinds: []string{
				KindCatalog, KindNamespace,
			},
			expectedCounts: map[string]int{
				KindCatalog:   2,
				KindNamespace: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := LoadResourceFromMultiYAMLFile(tt.filename, tt.yamlData)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Check that all expected kinds are present
			for _, kind := range tt.expectedKinds {
				assert.Contains(t, result, kind)
			}

			// Check that no unexpected kinds are present
			for kind := range result {
				assert.Contains(t, tt.expectedKinds, kind)
			}

			// Check resource counts
			for kind, expectedCount := range tt.expectedCounts {
				assert.Len(t, result[kind], expectedCount)
			}

			// Validate each resource structure
			for kind, resources := range result {
				assert.True(t, ValidateResourceKind(kind))

				for _, resource := range resources {
					// Check that JSON is valid
					assert.NotEmpty(t, resource.JSON)

					// Check metadata structure
					assert.Equal(t, kind, resource.Metadata.Kind)
					assert.NotNil(t, resource.Metadata.Metadata)

					// Verify JSON can be unmarshaled back to a map
					var jsonData map[string]any
					err := json.Unmarshal(resource.JSON, &jsonData)
					assert.NoError(t, err)

					// Check that kind and metadata are present in JSON
					assert.Equal(t, kind, jsonData["kind"])
					assert.Contains(t, jsonData, "metadata")
				}
			}
		})
	}
}

func TestLoadResourceFromMultiYAMLFile_ResourceContent(t *testing.T) {
	// Test specific content validation for a known file
	yamlData := []byte(`---
kind: Catalog
metadata:
  name: test-catalog
  labels:
    env: test
    version: v1
spec:
  description: A test catalog
  version: 1.0.0
---
kind: Namespace
metadata:
  name: test-namespace
  labels:
    env: prod
spec:
  description: A test namespace`)

	result, err := LoadResourceFromMultiYAMLFile("dummy.yaml", yamlData)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test Catalog resource content
	catalogs, exists := result[KindCatalog]
	assert.True(t, exists)
	assert.Len(t, catalogs, 1)

	catalog := catalogs[0]
	assert.Equal(t, KindCatalog, catalog.Metadata.Kind)

	// Check metadata content
	metadata := catalog.Metadata.Metadata
	assert.Equal(t, "test-catalog", metadata["name"])
	labels, ok := metadata["labels"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "test", labels["env"])
	assert.Equal(t, "v1", labels["version"])

	// Test Namespace resource content
	namespaces, exists := result[KindNamespace]
	assert.True(t, exists)
	assert.Len(t, namespaces, 1)

	namespace := namespaces[0]
	assert.Equal(t, KindNamespace, namespace.Metadata.Kind)

	// Check metadata content
	nsMetadata := namespace.Metadata.Metadata
	assert.Equal(t, "test-namespace", nsMetadata["name"])
	nsLabels, ok := nsMetadata["labels"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "prod", nsLabels["env"])
}

func TestLoadResourceFromMultiYAMLFile_JSONRoundTrip(t *testing.T) {
	// Test that JSON round-trip works correctly
	yamlData := []byte(`---
kind: Catalog
metadata:
  name: test-catalog
  labels:
    env: test
    version: v1
spec:
  description: A test catalog
  version: 1.0.0
---
kind: Namespace
metadata:
  name: test-namespace
  labels:
    env: prod
spec:
  description: A test namespace`)

	result, err := LoadResourceFromMultiYAMLFile("dummy.yaml", yamlData)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	for kind, resources := range result {
		for _, resource := range resources {
			// Unmarshal the JSON back to a map
			var jsonData map[string]any
			err := json.Unmarshal(resource.JSON, &jsonData)
			assert.NoError(t, err)

			// Verify the kind matches
			assert.Equal(t, kind, jsonData["kind"])

			// Verify metadata is present and matches
			metadata, exists := jsonData["metadata"]
			assert.True(t, exists)
			assert.Equal(t, resource.Metadata.Metadata, metadata)

			// Re-marshal and verify it matches the original JSON
			roundTripJSON, err := json.Marshal(jsonData)
			assert.NoError(t, err)
			assert.Equal(t, resource.JSON, roundTripJSON)
		}
	}
}
