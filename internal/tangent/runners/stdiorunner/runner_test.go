package stdiorunner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/internal/tangent/test"
	"github.com/tansive/tansive/pkg/api"
)

func TestNew(t *testing.T) {
	config.SetTestMode(true)
	test.SetupTest(t)
	config.TestInit(t)
	TestInit()
	tests := []struct {
		name       string
		jsonConfig json.RawMessage
		wantErr    bool
		errorType  apperrors.Error
		check      func(t *testing.T, r *runner)
	}{
		{
			name: "valid config",
			jsonConfig: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {
					"key1": "value1"
				},
				"env": {
					"VAR1": "value1"
				},
				"script": "test.sh",
				"security": {
					"type": "default"
				}
			}`, Version)),
			wantErr: false,
		},
		{
			name: "environment variables set correctly",
			jsonConfig: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {
					"TEST_VAR": "test_value",
					"FOO": "bar",
					"BAZ": "qux"
				},
				"script": "test.sh",
				"security": {
					"type": "default"
				}
			}`, Version)),
			wantErr: false,
			check: func(t *testing.T, r *runner) {
				assert.Equal(t, "test_value", r.config.Env["TEST_VAR"])
				assert.Equal(t, "bar", r.config.Env["FOO"])
				assert.Equal(t, "qux", r.config.Env["BAZ"])
			},
		},
		{
			name: "invalid runtime",
			jsonConfig: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "invalid",
				"runtimeConfig": {},
				"env": {},
				"script": "test.sh",
				"security": {
					"type": "default"
				}
			}`, Version)),
			wantErr:   true,
			errorType: ErrInvalidRuntime,
		},
		{
			name: "invalid security type",
			jsonConfig: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {},
				"script": "test.sh",
				"security": {
					"type": "invalid"
				}
			}`, Version)),
			wantErr:   true,
			errorType: ErrInvalidSecurity,
		},
		{
			name: "missing script",
			jsonConfig: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {},
				"security": {
					"type": "default"
				}
			}`, Version)),
			wantErr:   true,
			errorType: ErrInvalidScript,
		},
		{
			name: "incompatible version",
			jsonConfig: json.RawMessage(`{
				"version": "1.0.0",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {},
				"script": "test.sh",
				"security": {
					"type": "default"
				}
			}`),
			wantErr:   true,
			errorType: ErrInvalidVersion,
		},
		{
			name: "invalid version format",
			jsonConfig: json.RawMessage(`{
				"version": "invalid-version",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {},
				"script": "test.sh",
				"security": {
					"type": "default"
				}
			}`),
			wantErr:   true,
			errorType: ErrInvalidVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			writers := &tangentcommon.IOWriters{
				Out: io.Discard,
				Err: io.Discard,
			}

			var configMap map[string]any
			if err := json.Unmarshal(tt.jsonConfig, &configMap); err != nil {
				t.Fatalf("failed to unmarshal config: %v", err)
			}

			r, err := New(ctx, "test-session", configMap, writers)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
				assert.Nil(t, r)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, r)
				if tt.check != nil {
					tt.check(t, r)
				}
			}
		})
	}
}

func TestRun(t *testing.T) {
	config.SetTestMode(true)
	test.SetupTest(t)
	config.TestInit(t)
	TestInit()
	tests := []struct {
		name      string
		config    json.RawMessage
		args      json.RawMessage
		wantErr   bool
		errorType apperrors.Error
		check     func(t *testing.T, stdout, stderr string)
	}{
		{
			name: "successful execution",
			config: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {
					"TEST_VAR": "test_value"
				},
				"script": "test_script.sh",
				"security": {
					"type": "default"
				}
			}`, Version)),
			args: json.RawMessage(`{
				"arg1": "value1",
				"arg2": "value2"
			}`),
			wantErr: false,
		},
		{
			name: "environment variables passed correctly",
			config: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {
					"TEST_VAR": "test_value",
					"FOO": "bar",
					"BAZ": "qux"
				},
				"script": "test_script.sh",
				"security": {
					"type": "default"
				}
			}`, Version)),
			args: json.RawMessage(`{
				"arg1": "value1",
				"arg2": "value2",
				"check_env": true
			}`),
			wantErr: false,
			check: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stdout, "TEST_VAR=test_value")
				assert.Contains(t, stdout, "FOO=bar")
				assert.Contains(t, stdout, "BAZ=qux")

			},
		},
		{
			name: "script not found",
			config: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {
					"TEST_VAR": "test_value"
				},
				"script": "non_existent_script.sh",
				"security": {
					"type": "default"
				}
			}`, Version)),
			args: json.RawMessage(`{
				"arg1": "value1",
				"arg2": "value2"
			}`),
			wantErr:   true,
			errorType: ErrInvalidScript,
		},
		{
			name: "script execution failure",
			config: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {
					"TEST_VAR": "test_value"
				},
				"script": "test_script.sh",
				"security": {
					"type": "default"
				}
			}`, Version)),
			args: json.RawMessage(`{
				"arg1": "value1",
				"arg2": "value2",
				"should_fail": true
			}`),
			wantErr:   true,
			errorType: ErrExecutionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var stdout, stderr strings.Builder
			writers := &tangentcommon.IOWriters{
				Out: &stdout,
				Err: &stderr,
			}

			var configMap map[string]any
			if err := json.Unmarshal(tt.config, &configMap); err != nil {
				t.Fatalf("failed to unmarshal config: %v", err)
			}

			runner, err := New(ctx, "test-session", configMap, writers)
			require.NoError(t, err)
			require.NotNil(t, runner)

			var args map[string]any
			if err := json.Unmarshal(tt.args, &args); err != nil {
				t.Fatalf("failed to unmarshal args: %v", err)
			}
			skillArgs := &api.SkillInputArgs{
				InvocationID:     "test-invocation",
				SessionID:        "test-session",
				SkillName:        "test-skill",
				InputArgs:        args,
				SessionVariables: make(map[string]any),
			}
			err = runner.Run(ctx, skillArgs)
			t.Logf("stdout: %s", stdout.String())
			t.Logf("stderr: %s", stderr.String())
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, stdout.String(), stderr.String())
				}
			}
		})
	}
}

func TestDevModeSecurity(t *testing.T) {
	config.SetTestMode(true)
	test.SetupTest(t)
	config.TestInit(t)
	TestInit()
	tests := []struct {
		name      string
		config    json.RawMessage
		args      json.RawMessage
		wantErr   bool
		errorType apperrors.Error
		check     func(t *testing.T, stdout, stderr string)
	}{
		{
			name: "script path escaping attempt",
			config: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {
					"TEST_VAR": "test_value"
				},
				"script": "../../../etc/passwd",
				"security": {
					"type": "default"
				}
			}`, Version)),
			args: json.RawMessage(`{
				"arg1": "value1",
				"arg2": "value2"
			}`),
			wantErr:   true,
			errorType: ErrInvalidScript,
		},
		{
			name: "command injection attempt",
			config: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {
					"TEST_VAR": "test_value"
				},
				"script": "test_script.sh",
				"security": {
					"type": "default"
				}
			}`, Version)),
			args: json.RawMessage(`{
				"arg1": "value1; rm -rf /",
				"arg2": "value2; cat /etc/passwd"
			}`),
			wantErr: false,
			check: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stdout, "value1; rm -rf /")
				assert.Contains(t, stdout, "value2; cat /etc/passwd")
				assert.Contains(t, stdout, "Raw input:")
				assert.Contains(t, stdout, "Parsed JSON")
			},
		},
		{
			name: "home directory access attempt",
			config: json.RawMessage(fmt.Sprintf(`{
				"version": "%s",
				"runtime": "bash",
				"runtimeConfig": {},
				"env": {
					"TEST_VAR": "test_value"
				},
				"script": "test_script.sh",
				"security": {
					"type": "default"
				}
			}`, Version)),
			args: json.RawMessage(`{
				"arg1": "value1",
				"arg2": "value2",
				"check_home": true
			}`),
			wantErr: false,
			check: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stdout, "Home directory contents:")
				assert.Contains(t, stdout, "total")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var stdout, stderr strings.Builder
			writers := &tangentcommon.IOWriters{
				Out: &stdout,
				Err: &stderr,
			}

			var configMap map[string]any
			if err := json.Unmarshal(tt.config, &configMap); err != nil {
				t.Fatalf("failed to unmarshal config: %v", err)
			}

			runner, err := New(ctx, "test-session", configMap, writers)
			require.NoError(t, err)
			require.NotNil(t, runner)

			var args map[string]any
			if err := json.Unmarshal(tt.args, &args); err != nil {
				t.Fatalf("failed to unmarshal args: %v", err)
			}
			skillArgs := &api.SkillInputArgs{
				InvocationID:     "test-invocation",
				SessionID:        "test-session",
				SkillName:        "test-skill",
				InputArgs:        args,
				SessionVariables: make(map[string]any),
			}
			err = runner.Run(ctx, skillArgs)
			t.Logf("stdout: %s", stdout.String())
			t.Logf("stderr: %s", stderr.String())
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, stdout.String(), stderr.String())
				}
			}
		})
	}
}

func TestNormalizeLineEndings(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "lineending_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test script with Windows line endings
	sourcePath := filepath.Join(tempDir, "test_script.sh")
	windowsContent := "#!/bin/bash\r\nset -e\r\necho 'Hello World'\r\n"
	err = os.WriteFile(sourcePath, []byte(windowsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Create a runner instance for testing
	r := &runner{}

	// Test line ending normalization
	targetPath := filepath.Join(tempDir, "normalized_script.sh")
	err = r.normalizeLineEndings(sourcePath, targetPath)
	if err != nil {
		t.Fatalf("Failed to normalize line endings: %v", err)
	}

	// Read the normalized content
	normalizedContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read normalized file: %v", err)
	}

	// Verify that \r\n has been converted to \n
	expectedContent := "#!/bin/bash\nset -e\necho 'Hello World'\n"
	if string(normalizedContent) != expectedContent {
		t.Errorf("Line ending normalization failed. Expected: %q, Got: %q", expectedContent, string(normalizedContent))
	}
}

func TestShouldNormalizeLineEndings(t *testing.T) {
	r := &runner{}

	tests := []struct {
		name       string
		scriptPath string
		runtime    Runtime
		expected   bool
	}{
		{
			name:       "bash runtime should normalize",
			scriptPath: "script.py",
			runtime:    RuntimeBash,
			expected:   true,
		},
		{
			name:       "sh extension should normalize",
			scriptPath: "script.sh",
			runtime:    RuntimePython,
			expected:   true,
		},
		{
			name:       "bash extension should normalize",
			scriptPath: "script.bash",
			runtime:    RuntimeNode,
			expected:   true,
		},
		{
			name:       "py extension should normalize",
			scriptPath: "script.py",
			runtime:    RuntimePython,
			expected:   true,
		},
		{
			name:       "js extension should normalize",
			scriptPath: "script.js",
			runtime:    RuntimeNode,
			expected:   true,
		},
		{
			name:       "txt extension should not normalize",
			scriptPath: "script.txt",
			runtime:    RuntimePython,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.config.Runtime = tt.runtime
			result := r.shouldNormalizeLineEndings(tt.scriptPath)
			if result != tt.expected {
				t.Errorf("shouldNormalizeLineEndings(%s, %s) = %v, want %v", tt.scriptPath, tt.runtime, result, tt.expected)
			}
		})
	}
}
