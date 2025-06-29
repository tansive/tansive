package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ResourceMetadata represents a generic resource with Kind and metadata
type ResourceMetadata struct {
	Kind     string         `json:"kind" yaml:"kind"`
	Metadata map[string]any `json:"metadata" yaml:"metadata"`
}

type Resource struct {
	JSON     []byte
	Metadata ResourceMetadata
}

type ResourceList []Resource

// LoadResourceFromFile loads a resource from a YAML file and converts it to JSON
// Returns the JSON data and a parsed Resource struct
func LoadResourceFromFile(filename string) ([]byte, *ResourceMetadata, error) {
	// Read the YAML file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Remove stray tabs
	data = replaceTabsWithSpaces(data)

	data, err = PreprocessYAML(data)
	if err != nil {
		return nil, nil, err
	}

	// Decode YAML into a generic interface
	var yamlData map[string]any
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return nil, nil, fmt.Errorf("unable to parse YAML: %v", err)
	}

	// Encode the decoded data to JSON
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to convert to JSON: %v", err)
	}

	var resource ResourceMetadata
	if err := json.Unmarshal(jsonData, &resource); err != nil {
		return nil, nil, fmt.Errorf("failed to parse resource: %v", err)
	}

	return jsonData, &resource, nil
}

// LoadResourceFromMultiYAMLFile loads a resource from a multi-YAML file
// Returns a slice of maps containing the parsed YAML documents
// If data is provided, it will be used instead of reading from the file
func LoadResourceFromMultiYAMLFile(filename string, data ...[]byte) (map[string]ResourceList, error) {
	var yamlData []byte
	var err error

	if len(data) > 0 {
		// Use provided data instead of reading from file
		yamlData = data[0]
	} else {
		// Read the YAML file
		yamlData, err = os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
	}

	// Remove stray tabs
	yamlData = replaceTabsWithSpaces(yamlData)

	yamlData, err = PreprocessYAML(yamlData)
	if err != nil {
		return nil, err
	}

	resources, err := ParseMultiYAMLFromBytes(yamlData)
	if err != nil {
		return nil, err
	}

	result := make(map[string]ResourceList)

	for _, resource := range resources {
		kind, ok := resource["kind"].(string)
		if !ok {
			return nil, fmt.Errorf("resource kind: %v is not a string", resource["kind"])
		}
		if !ValidateResourceKind(kind) {
			return nil, fmt.Errorf("invalid resource kind: %s", kind)
		}

		metadataAny, exists := resource["metadata"]
		if !exists {
			return nil, fmt.Errorf("metadata not found in resource: %v", kind)
		}
		metadata, ok := metadataAny.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("metadata has invalid format: %v", metadataAny)
		}

		jsonData, err := json.Marshal(resource)
		if err != nil {
			return nil, fmt.Errorf("unable to parse resource: %v", err)
		}

		result[kind] = append(result[kind], Resource{
			JSON: jsonData,
			Metadata: ResourceMetadata{
				Kind:     kind,
				Metadata: metadata,
			},
		})
	}

	return result, nil
}

// replaceTabsWithSpaces replaces all tab characters with four spaces in a byte slice
func replaceTabsWithSpaces(b []byte) []byte {
	space := []byte("    ")
	var result []byte
	for _, c := range b {
		if c == '\t' {
			result = append(result, space...)
		} else {
			result = append(result, c)
		}
	}
	return result
}

// GetResourceType returns the API endpoint path for a given resource kind
// Maps resource kinds to their corresponding API endpoints
func GetResourceType(kind string) (string, error) {
	switch kind {
	case KindCatalog:
		return "catalogs", nil
	case KindVariant:
		return "variants", nil
	case KindNamespace:
		return "namespaces", nil
	case KindView:
		return "views", nil
	case KindResource:
		return "resources", nil
	case KindSkillset:
		return "skillsets", nil
	default:
		return "", fmt.Errorf("unknown resource kind: %s", kind)
	}
}

// MapResourceTypeToURL maps a resource type string to its URL format
// Handles various aliases for each resource type
func MapResourceTypeToURL(resourceType string) (string, error) {
	switch resourceType {
	case "catalog", "cat", "catalogs":
		return "catalogs", nil
	case "variant", "var", "variants":
		return "variants", nil
	case "namespace", "ns", "namespaces":
		return "namespaces", nil
	case "view", "v", "views":
		return "views", nil
	case "resource", "res", "resources":
		return "resources", nil
	case "skillset", "sk", "skillsets":
		return "skillsets", nil
	case "session", "sess", "sessions":
		return "sessions", nil
	default:
		return "", fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

func StrictMapAnyAnyToStringAny(input any) (map[string]any, error) {
	converted, err := convertRecursively(input)
	if err != nil {
		return nil, err
	}
	result, ok := converted.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected top-level object to be a map[string]any, got %T", converted)
	}
	return result, nil
}

func convertRecursively(input any) (any, error) {
	switch v := input.(type) {
	case map[any]any:
		result := make(map[string]any)
		for k, val := range v {
			strKey, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string map key: %v (type %T)", k, k)
			}
			convertedVal, err := convertRecursively(val)
			if err != nil {
				return nil, err
			}
			result[strKey] = convertedVal
		}
		return result, nil

	case []any:
		for i, elem := range v {
			convertedElem, err := convertRecursively(elem)
			if err != nil {
				return nil, err
			}
			v[i] = convertedElem
		}
		return v, nil

	default:
		return v, nil
	}
}
