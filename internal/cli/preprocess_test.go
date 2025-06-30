package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPreprocessYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		envFile  string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple environment variable substitution",
			input:    "api_key: {{ .ENV.API_KEY }}",
			envVars:  map[string]string{"API_KEY": "secret123"},
			expected: "api_key: secret123",
			wantErr:  false,
		},
		{
			name:     "multiple environment variables",
			input:    "host: {{ .ENV.HOST }}\nport: {{ .ENV.PORT }}",
			envVars:  map[string]string{"HOST": "localhost", "PORT": "8080"},
			expected: "host: localhost\nport: 8080",
			wantErr:  false,
		},
		{
			name:     "environment variable with special characters",
			input:    "password: {{ .ENV.DB_PASSWORD }}",
			envVars:  map[string]string{"DB_PASSWORD": "p@ssw0rd!@#"},
			expected: "password: p@ssw0rd!@#",
			wantErr:  false,
		},
		{
			name:     "empty environment variable",
			input:    "empty: {{ .ENV.EMPTY_VAR }}",
			envVars:  map[string]string{"EMPTY_VAR": ""},
			expected: "empty: ",
			wantErr:  false,
		},
		{
			name:     "no template variables",
			input:    "simple: yaml\ncontent: here",
			envVars:  map[string]string{},
			expected: "simple: yaml\ncontent: here",
			wantErr:  false,
		},
		{
			name:     "missing environment variable should error",
			input:    "missing: {{ .ENV.MISSING_VAR }}",
			envVars:  map[string]string{},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "invalid template syntax",
			input:    "invalid: {{ .ENV.VAR }",
			envVars:  map[string]string{"VAR": "value"},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "complex YAML with mixed content",
			input:    "api:\n  key: {{ .ENV.API_KEY }}\n  url: {{ .ENV.API_URL }}\ndatabase:\n  host: {{ .ENV.DB_HOST }}",
			envVars:  map[string]string{"API_KEY": "secret", "API_URL": "https://api.example.com", "DB_HOST": "localhost"},
			expected: "api:\n  key: secret\n  url: https://api.example.com\ndatabase:\n  host: localhost",
			wantErr:  false,
		},
		{
			name:     "environment variable with equals sign in value",
			input:    "config: {{ .ENV.CONFIG_VAR }}",
			envVars:  map[string]string{"CONFIG_VAR": "key=value"},
			expected: "config: key=value",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Create .env file if specified
			if tt.envFile != "" {
				envPath := filepath.Join(t.TempDir(), ".env")
				err := os.WriteFile(envPath, []byte(tt.envFile), 0644)
				if err != nil {
					t.Fatalf("Failed to create .env file: %v", err)
				}

				// Change to the temp directory temporarily
				originalWd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get current working directory: %v", err)
				}
				defer os.Chdir(originalWd)

				err = os.Chdir(filepath.Dir(envPath))
				if err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			}

			// Run the function
			result, err := PreprocessYAML([]byte(tt.input))

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("PreprocessYAML() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("PreprocessYAML() unexpected error: %v", err)
				return
			}

			// Check result
			if string(result) != tt.expected {
				t.Errorf("PreprocessYAML() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestPreprocessYAMLWithEnvFile(t *testing.T) {
	// Create a temporary directory for .env file
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to temp directory
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create .env file
	envContent := `API_KEY=from_env_file
DB_HOST=localhost
DB_PORT=5432`
	err = os.WriteFile(".env", []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Set some environment variables that should override .env file
	os.Setenv("API_KEY", "from_environment")
	defer os.Unsetenv("API_KEY")

	input := `api:
  key: {{ .ENV.API_KEY }}
database:
  host: {{ .ENV.DB_HOST }}
  port: {{ .ENV.DB_PORT }}`

	expected := `api:
  key: from_environment
database:
  host: localhost
  port: 5432`

	result, err := PreprocessYAML([]byte(input))
	if err != nil {
		t.Errorf("PreprocessYAML() unexpected error: %v", err)
		return
	}

	if string(result) != expected {
		t.Errorf("PreprocessYAML() = %q, want %q", string(result), expected)
	}
}

func TestPreprocessYAMLNoEnvFile(t *testing.T) {
	// Test when no .env file exists
	input := "test: {{ .ENV.TEST_VAR }}"
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	expected := "test: test_value"

	result, err := PreprocessYAML([]byte(input))
	if err != nil {
		t.Errorf("PreprocessYAML() unexpected error: %v", err)
		return
	}

	if string(result) != expected {
		t.Errorf("PreprocessYAML() = %q, want %q", string(result), expected)
	}
}

func TestPreprocessYAMLEnvironmentVariableParsing(t *testing.T) {
	// Test that environment variables with equals signs in values are handled correctly
	os.Setenv("COMPLEX_VAR", "key=value&another=thing")
	defer os.Unsetenv("COMPLEX_VAR")

	input := "config: {{ .ENV.COMPLEX_VAR }}"
	expected := "config: key=value&another=thing"

	result, err := PreprocessYAML([]byte(input))
	if err != nil {
		t.Errorf("PreprocessYAML() unexpected error: %v", err)
		return
	}

	if string(result) != expected {
		t.Errorf("PreprocessYAML() = %q, want %q", string(result), expected)
	}
}

func TestPreprocessYAMLEmptyInput(t *testing.T) {
	result, err := PreprocessYAML([]byte(""))
	if err != nil {
		t.Errorf("PreprocessYAML() unexpected error: %v", err)
		return
	}

	if string(result) != "" {
		t.Errorf("PreprocessYAML() = %q, want empty string", string(result))
	}
}

func TestPreprocessYAMLTemplateContext(t *testing.T) {
	// Test that the TemplateContext struct is properly used
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	input := "{{ .ENV.TEST_KEY }}"
	expected := "test_value"

	result, err := PreprocessYAML([]byte(input))
	if err != nil {
		t.Errorf("PreprocessYAML() unexpected error: %v", err)
		return
	}

	if string(result) != expected {
		t.Errorf("PreprocessYAML() = %q, want %q", string(result), expected)
	}
}

// Test cases that could potentially cause panics
func TestPreprocessYAMLPanicScenarios(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		envVars     map[string]string
		description string
	}{
		{
			name:        "very large input",
			input:       strings.Repeat("{{ .ENV.TEST_VAR }}", 10000),
			envVars:     map[string]string{"TEST_VAR": "value"},
			description: "Tests handling of very large template inputs",
		},
		{
			name:        "deeply nested template",
			input:       "{{ .ENV.{{ .ENV.NESTED }}}}",
			envVars:     map[string]string{"NESTED": "TEST_VAR", "TEST_VAR": "value"},
			description: "Tests nested template expressions",
		},
		{
			name:        "template with function calls",
			input:       "{{ .ENV.TEST_VAR | len }}",
			envVars:     map[string]string{"TEST_VAR": "value"},
			description: "Tests template with function calls",
		},
		{
			name:        "malformed environment variable name",
			input:       "{{ .ENV.INVALID-NAME }}",
			envVars:     map[string]string{},
			description: "Tests malformed environment variable names",
		},
		{
			name:        "template with range",
			input:       "{{ range .ENV }}{{ . }}{{ end }}",
			envVars:     map[string]string{"TEST": "value"},
			description: "Tests template with range loops",
		},
		{
			name:        "template with conditional",
			input:       "{{ if .ENV.TEST_VAR }}{{ .ENV.TEST_VAR }}{{ end }}",
			envVars:     map[string]string{"TEST_VAR": "value"},
			description: "Tests template with conditionals",
		},
		{
			name:        "template with index access",
			input:       "{{ index .ENV \"TEST_VAR\" }}",
			envVars:     map[string]string{"TEST_VAR": "value"},
			description: "Tests template with index access",
		},
		{
			name:        "template with nil access",
			input:       "{{ .ENV.NIL_VAR }}",
			envVars:     map[string]string{},
			description: "Tests access to non-existent environment variable",
		},
		{
			name:        "template with empty key",
			input:       "{{ .ENV. }}",
			envVars:     map[string]string{},
			description: "Tests template with empty key",
		},
		{
			name:        "template with special characters in key",
			input:       "{{ .ENV.TEST_VAR_WITH_SPECIAL_CHARS_!@#$%^&*() }}",
			envVars:     map[string]string{"TEST_VAR_WITH_SPECIAL_CHARS_!@#$%^&*()": "value"},
			description: "Tests template with special characters in environment variable names",
		},
		{
			name:        "template with unicode characters",
			input:       "{{ .ENV.UNICODE_VAR }}",
			envVars:     map[string]string{"UNICODE_VAR": "🚀🌟✨"},
			description: "Tests template with unicode characters",
		},
		{
			name:        "template with very long environment variable value",
			input:       "{{ .ENV.LONG_VAR }}",
			envVars:     map[string]string{"LONG_VAR": strings.Repeat("very_long_value_", 1000)},
			description: "Tests template with very long environment variable values",
		},
		{
			name:        "template with newlines in environment variable",
			input:       "{{ .ENV.MULTILINE_VAR }}",
			envVars:     map[string]string{"MULTILINE_VAR": "line1\nline2\nline3"},
			description: "Tests template with newlines in environment variable values",
		},
		{
			name:        "template with null bytes",
			input:       "{{ .ENV.NULL_VAR }}",
			envVars:     map[string]string{"NULL_VAR": "value\x00with\x00nulls"},
			description: "Tests template with null bytes in environment variable values",
		},
		{
			name:        "template with control characters",
			input:       "{{ .ENV.CONTROL_VAR }}",
			envVars:     map[string]string{"CONTROL_VAR": "value\x01\x02\x03"},
			description: "Tests template with control characters in environment variable values",
		},
		{
			name:        "template with recursive reference",
			input:       "{{ .ENV.RECURSIVE_VAR }}",
			envVars:     map[string]string{"RECURSIVE_VAR": "{{ .ENV.RECURSIVE_VAR }}"},
			description: "Tests template with recursive environment variable references",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Use defer recover to catch any panics (function should never panic)
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PreprocessYAML() panicked unexpectedly: %v", r)
				}
			}()

			// Run the function
			result, err := PreprocessYAML([]byte(tt.input))

			// The function should never panic, so we expect either success or a proper error
			if err != nil {
				t.Logf("PreprocessYAML() returned expected error: %v", err)
				return
			}

			// Log the result for debugging
			t.Logf("PreprocessYAML() succeeded with result length: %d", len(result))
		})
	}
}

func TestPreprocessYAMLMalformedEnvironmentVariables(t *testing.T) {
	// Test cases for malformed environment variables that could cause issues
	tests := []struct {
		name        string
		envVar      string
		envValue    string
		description string
	}{
		{
			name:        "environment variable with no equals sign",
			envVar:      "MALFORMED_VAR",
			envValue:    "no_equals_sign",
			description: "Tests environment variable without equals sign",
		},
		{
			name:        "environment variable with multiple equals signs",
			envVar:      "MULTI_EQUALS",
			envValue:    "key=value=another",
			description: "Tests environment variable with multiple equals signs",
		},
		{
			name:        "environment variable with only equals sign",
			envVar:      "",
			envValue:    "=value",
			description: "Tests environment variable with empty key",
		},
		{
			name:        "environment variable with empty value",
			envVar:      "EMPTY_VALUE",
			envValue:    "",
			description: "Tests environment variable with empty value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the malformed environment variable
			if tt.envVar != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			// Use defer recover to catch any panics
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PreprocessYAML() panicked with malformed env var: %v", r)
				}
			}()

			// Test with a simple template
			input := "test: {{ .ENV.TEST_VAR }}"
			os.Setenv("TEST_VAR", "normal_value")
			defer os.Unsetenv("TEST_VAR")

			result, err := PreprocessYAML([]byte(input))
			if err != nil {
				t.Logf("PreprocessYAML() returned expected error: %v", err)
				return
			}

			expected := "test: normal_value"
			if string(result) != expected {
				t.Errorf("PreprocessYAML() = %q, want %q", string(result), expected)
			}
		})
	}
}

func TestPreprocessYAMLExtremeCases(t *testing.T) {
	// Test extreme cases that could cause memory issues or panics
	tests := []struct {
		name        string
		input       string
		envVars     map[string]string
		description string
	}{
		{
			name:        "extremely large template",
			input:       strings.Repeat("{{ .ENV.TEST_VAR }}", 100000),
			envVars:     map[string]string{"TEST_VAR": "value"},
			description: "Tests extremely large template input",
		},
		{
			name: "template with many environment variables",
			input: func() string {
				var parts []string
				for i := 0; i < 1000; i++ {
					parts = append(parts, "{{ .ENV.VAR_"+string(rune(i+'0'))+" }}")
				}
				return strings.Join(parts, "\n")
			}(),
			envVars: func() map[string]string {
				m := make(map[string]string)
				for i := 0; i < 1000; i++ {
					m["VAR_"+string(rune(i+'0'))] = "value"
				}
				return m
			}(),
			description: "Tests template with many environment variable references",
		},
		{
			name:        "template with deeply nested braces",
			input:       strings.Repeat("{{", 1000) + " .ENV.TEST_VAR " + strings.Repeat("}}", 1000),
			envVars:     map[string]string{"TEST_VAR": "value"},
			description: "Tests template with deeply nested braces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Use defer recover to catch any panics
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PreprocessYAML() panicked in extreme case: %v", r)
				}
			}()

			// Run the function with timeout protection
			done := make(chan bool, 1)
			var result []byte
			var err error

			go func() {
				result, err = PreprocessYAML([]byte(tt.input))
				done <- true
			}()

			select {
			case <-done:
				// Function completed
				if err != nil {
					t.Logf("PreprocessYAML() returned expected error: %v", err)
					return
				}
				t.Logf("PreprocessYAML() succeeded with result length: %d", len(result))
			case <-time.After(5 * time.Second):
				t.Errorf("PreprocessYAML() timed out after 5 seconds")
			}
		})
	}
}
