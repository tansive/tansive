package schemavalidator

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestResourcePathValidator(t *testing.T) {
	validate := validator.New()
	validate.RegisterValidation("resourcepath", resourcePathValidator)

	tests := []struct {
		input   string
		isValid bool
	}{
		{input: "/valid-path/with-collections", isValid: true},
		{input: "/valid-collection", isValid: true},
		{input: "/invalid-path/with@chars", isValid: false},
		{input: "relative/path", isValid: false},
		{input: "/another-valid-collection/", isValid: true},
		{input: "/collection_with_underscore/anotherCollection", isValid: false},
		{input: "/invalid-collection//double-slash", isValid: true},
		{input: "/", isValid: true},
		{input: "", isValid: false},
	}

	for _, test := range tests {
		err := validate.Var(test.input, "resourcepath")
		if (err == nil) != test.isValid {
			t.Errorf("Expected %v for input '%s', but got %v", test.isValid, test.input, err == nil)
		}
	}
}

func TestNoSpacesValidator(t *testing.T) {
	validate := validator.New()
	validate.RegisterValidation("noSpaces", noSpacesValidator)

	// Test cases
	tests := []struct {
		input    string
		expected bool
	}{
		{"ValidString", true},        // No spaces, valid string
		{"Invalid String", false},    // Contains spaces
		{"Invalid\tTab", false},      // Contains tab
		{"Invalid\nNewline", false},  // Contains newline
		{"AnotherValidString", true}, // No spaces
		{"", false},                  // Empty string, should fail
		{"Multiple   Spaces", false}, // Multiple spaces
	}

	for _, test := range tests {
		err := validate.Var(test.input, "noSpaces")
		result := err == nil

		if result != test.expected {
			t.Errorf("Expected %v for input '%s', got %v", test.expected, test.input, result)
		}
	}
}
