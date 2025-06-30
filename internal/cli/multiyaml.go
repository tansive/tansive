package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseMultiYAML parses a file containing multiple YAML documents
// Returns a slice of maps containing the parsed YAML documents
func ParseMultiYAML(filename string) ([]map[string]any, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	data = replaceTabsWithSpaces(data)

	data, err = PreprocessYAML(data)
	if err != nil {
		return nil, err
	}

	return ParseMultiYAMLFromBytes(data)
}

// ParseMultiYAMLFromBytes parses byte data containing multiple YAML documents
// Returns a slice of maps containing the parsed YAML documents
func ParseMultiYAMLFromBytes(data []byte) ([]map[string]any, error) {
	// If data is empty or contains only whitespace or only --- separators, return empty slice
	content := strings.TrimSpace(string(data))
	if len(content) == 0 || strings.Trim(content, "- \n\t") == "" {
		return []map[string]any{}, nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var result []map[string]any

	for {
		var doc map[string]any
		if err := decoder.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}
		// Skip empty documents (common with trailing ---)
		if len(doc) > 0 {
			result = append(result, doc)
		}
	}

	return result, nil
}
