package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMultiYAML(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "multiyaml-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		content  string
		expected []map[string]any
		wantErr  bool
	}{
		{
			name: "valid multi-document YAML",
			content: `---
name: doc1
value: 1
---
name: doc2
value: 2`,
			expected: []map[string]any{
				{"name": "doc1", "value": 1},
				{"name": "doc2", "value": 2},
			},
			wantErr: false,
		},
		{
			name: "documents without leading ---",
			content: `name: doc1
value: 1
---
name: doc2
value: 2`,
			expected: []map[string]any{
				{"name": "doc1", "value": 1},
				{"name": "doc2", "value": 2},
			},
			wantErr: false,
		},
		{
			name: "single document without separators",
			content: `name: single_doc
value: 42
nested:
  key: value
  list: [1, 2, 3]`,
			expected: []map[string]any{
				{
					"name":  "single_doc",
					"value": 42,
					"nested": map[string]any{
						"key":  "value",
						"list": []any{1, 2, 3},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "one invalid document in multi-document YAML",
			content: `---
name: valid_doc
value: 1
---
invalid: yaml: content: with: colons: everywhere
---
name: another_valid_doc
value: 2`,
			expected: nil,
			wantErr:  true,
		},
		{
			name: "empty document handling",
			content: `---
name: doc1
---
---
name: doc2`,
			expected: []map[string]any{
				{"name": "doc1"},
				{"name": "doc2"},
			},
			wantErr: false,
		},
		{
			name:     "completely empty file",
			content:  ``,
			expected: []map[string]any{},
			wantErr:  false,
		},
		{
			name:     "file with only whitespace",
			content:  "   \n\t  \n",
			expected: []map[string]any{},
			wantErr:  false,
		},
		{
			name:     "file with only document separators",
			content:  "---\n---\n---",
			expected: []map[string]any{},
			wantErr:  false,
		},
		{
			name:     "invalid YAML",
			content:  `invalid: yaml: content:`,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file with test content
			tmpFile := filepath.Join(tmpDir, "test.yaml")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			assert.NoError(t, err)

			// Test ParseMultiYAML
			result, err := ParseMultiYAML(tmpFile)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}

	// Test file not found
	t.Run("file not found", func(t *testing.T) {
		result, err := ParseMultiYAML("nonexistent.yaml")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
