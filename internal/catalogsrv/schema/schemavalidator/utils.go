package schemavalidator

import (
	"reflect"
	"strings"
)

// GetJSONTag retrieves the JSON tag for a given struct field.
// If the JSON tag is not found or is explicitly ignored, it falls back to the field name.
func GetJSONTag(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return field.Name
	}
	return strings.Split(jsonTag, ",")[0]
}

// GetJSONFieldPath recursively fetches the JSON field path for a given field in a struct.
// It iterates through all the fields in the struct, checking if each field matches the target field name.
// If the field is a nested struct, it recurses into its fields to find the target field name and constructs the full JSON path.
func GetJSONFieldPath(structVal reflect.Value, structType reflect.Type, fieldName string) string {
	// Iterate through all the fields in the struct
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structVal.Field(i)

		// Check if this field matches the target field name
		if field.Name == fieldName {
			return GetJSONTag(field)
		}

		// If the field is a nested struct, recurse into its fields
		if field.Type.Kind() == reflect.Struct {
			nestedPath := GetJSONFieldPath(fieldValue, field.Type, fieldName)
			if nestedPath != "" {
				return GetJSONTag(field) + "." + nestedPath
			}
		}

		// If the field is a pointer to a struct, dereference it
		if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
			if !fieldValue.IsNil() { // Check if the pointer is not nil
				// Dereference the pointer to get the actual struct value
				dereferencedValue := fieldValue.Elem()
				nestedPath := GetJSONFieldPath(dereferencedValue, dereferencedValue.Type(), fieldName)
				if nestedPath != "" {
					return GetJSONTag(field) + "." + nestedPath
				}
			}
		}

		// If the field is a map and the key type is string, check the values
		if field.Type.Kind() == reflect.Map && field.Type.Key().Kind() == reflect.String {
			// Iterate through the map keys and values
			for _, key := range fieldValue.MapKeys() {
				mapValue := fieldValue.MapIndex(key)

				// If the map value is a struct, recurse into it
				if mapValue.Kind() == reflect.Struct {
					nestedPath := GetJSONFieldPath(mapValue, mapValue.Type(), fieldName)
					if nestedPath != "" {
						// Return the JSON tag for the map field, the actual map key value, and the nested path
						return GetJSONTag(field) + "." + key.String() + "." + nestedPath
					}
				}
			}
		}
	}

	// If no match was found, return an empty string
	return ""
}
