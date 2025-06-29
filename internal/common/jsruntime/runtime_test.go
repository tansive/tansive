package jsruntime

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"errors"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		jsCode  string
		wantErr bool
	}{
		{
			name:    "valid function",
			jsCode:  "function(a, b) { return a + b; }",
			wantErr: false,
		},
		{
			name:    "valid arrow function",
			jsCode:  "(a, b) => a + b",
			wantErr: false,
		},
		{
			name:    "valid function with complex logic",
			jsCode:  "function(session, input) { return { sum: session.value + input.value, timestamp: Date.now() }; }",
			wantErr: false,
		},
		{
			name:    "invalid syntax",
			jsCode:  "function(a, b { return a + b; }", // missing closing parenthesis
			wantErr: true,
		},
		{
			name:    "not a function",
			jsCode:  "var x = 42;",
			wantErr: true,
		},
		{
			name:    "empty string",
			jsCode:  "",
			wantErr: true,
		},
		{
			name:    "just whitespace",
			jsCode:  "   \n\t  ",
			wantErr: true,
		},
		{
			name:    "function with console.log",
			jsCode:  "function(a, b) { console.log('test'); return a + b; }",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsFunc, err := New(context.Background(), tt.jsCode)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, jsFunc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, jsFunc)
				assert.Equal(t, tt.jsCode, jsFunc.code)
				assert.NotNil(t, jsFunc.function)
			}
		})
	}
}

func TestJSFunction_Run(t *testing.T) {
	tests := []struct {
		name        string
		jsCode      string
		sessionArgs map[string]any
		inputArgs   map[string]any
		timeout     time.Duration
		wantResult  map[string]any
		wantErr     bool
	}{
		{
			name:        "simple addition",
			jsCode:      "function(a, b) { return { result: a.value + b.value }; }",
			sessionArgs: map[string]any{"value": 5},
			inputArgs:   map[string]any{"value": 3},
			wantResult:  map[string]any{"result": int64(8)},
			wantErr:     false,
		},
		{
			name:        "object manipulation",
			jsCode:      "function(session, input) { return { result: session.value + input.value, session: session, input: input }; }",
			sessionArgs: map[string]any{"value": 10, "id": "session1"},
			inputArgs:   map[string]any{"value": 20, "id": "input1"},
			wantResult:  map[string]any{"input": map[string]any{"id": "input1", "value": 20}, "result": int64(30), "session": map[string]any{"id": "session1", "value": 10}},
			wantErr:     false,
		},
		{
			name:        "array operations",
			jsCode:      "function(session, input) { return { result: session.items.concat(input.items) }; }",
			sessionArgs: map[string]any{"items": []any{1, 2, 3}},
			inputArgs:   map[string]any{"items": []any{4, 5, 6}},
			wantResult:  map[string]any{"result": []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}},
			wantErr:     false,
		},
		{
			name:        "conditional logic",
			jsCode:      "function(session, input) { return { result: session.enabled ? input.value : 0 }; }",
			sessionArgs: map[string]any{"enabled": true},
			inputArgs:   map[string]any{"value": 42},
			wantResult:  map[string]any{"result": int64(42)},
			wantErr:     false,
		},
		{
			name:        "conditional logic false",
			jsCode:      "function(session, input) { return { result: session.enabled ? input.value : 0 }; }",
			sessionArgs: map[string]any{"enabled": false},
			inputArgs:   map[string]any{"value": 42},
			wantResult:  map[string]any{"result": int64(0)},
			wantErr:     false,
		},
		{
			name:        "null and undefined handling",
			jsCode:      "function(session, input) { return { sessionNull: session === null, inputUndefined: input === undefined, sessionType: typeof session, inputType: typeof input }; }",
			sessionArgs: nil,
			inputArgs:   nil,
			wantResult:  map[string]any{"inputType": "object", "inputUndefined": false, "sessionNull": true, "sessionType": "object"},
			wantErr:     false,
		},
		{
			name:        "empty objects",
			jsCode:      "function(session, input) { return { count: Object.keys(session).length + Object.keys(input).length }; }",
			sessionArgs: map[string]any{},
			inputArgs:   map[string]any{},
			wantResult:  map[string]any{"count": int64(0)},
			wantErr:     false,
		},
		{
			name:        "complex nested objects",
			jsCode:      "function(session, input) { return { deep: { nested: { value: session.config.deep.nested.value + input.config.deep.nested.value } } }; }",
			sessionArgs: map[string]any{"config": map[string]any{"deep": map[string]any{"nested": map[string]any{"value": 100}}}},
			inputArgs:   map[string]any{"config": map[string]any{"deep": map[string]any{"nested": map[string]any{"value": 200}}}},
			wantResult:  map[string]any{"deep": map[string]any{"nested": map[string]any{"value": int64(300)}}},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsFunc, err := New(context.Background(), tt.jsCode)
			require.NoError(t, err)

			opts := Options{Timeout: tt.timeout}
			if opts.Timeout == 0 {
				opts.Timeout = 100 * time.Millisecond
			}

			result, err := jsFunc.Run(context.Background(), tt.sessionArgs, tt.inputArgs, opts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

func TestJSFunction_Run_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		jsCode      string
		sessionArgs map[string]any
		inputArgs   map[string]any
		timeout     time.Duration
		expectedErr error
	}{
		{
			name:        "runtime error in function",
			jsCode:      "function(a, b) { return a.nonExistentProperty.method(); }",
			sessionArgs: map[string]any{"value": 5},
			inputArgs:   map[string]any{"value": 3},
			expectedErr: ErrJSRuntimeError,
		},
		{
			name:        "reference error",
			jsCode:      "function(a, b) { return undefinedVariable; }",
			sessionArgs: map[string]any{"value": 5},
			inputArgs:   map[string]any{"value": 3},
			expectedErr: ErrJSRuntimeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsFunc, err := New(context.Background(), tt.jsCode)
			require.NoError(t, err)

			opts := Options{Timeout: tt.timeout}
			if opts.Timeout == 0 {
				opts.Timeout = 100 * time.Millisecond
			}

			result, err := jsFunc.Run(context.Background(), tt.sessionArgs, tt.inputArgs, opts)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.True(t, errors.Is(err, tt.expectedErr), "expected error to be %v, got %v", tt.expectedErr, err)
		})
	}
}

func TestJSFunction_Run_Timeout(t *testing.T) {
	// Function that runs indefinitely
	jsCode := "function(a, b) { while(true) { } return a + b; }"

	jsFunc, err := New(context.Background(), jsCode)
	require.NoError(t, err)

	opts := Options{Timeout: 10 * time.Millisecond}

	start := time.Now()
	result, err := jsFunc.Run(context.Background(), map[string]any{"value": 5}, map[string]any{"value": 3}, opts)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "timeout")
	assert.Less(t, duration, 50*time.Millisecond) // Should timeout quickly
}

func TestJSFunction_Run_DefaultTimeout(t *testing.T) {
	jsCode := "function(a, b) { return { result: a.value + b.value }; }"
	jsFunc, err := New(context.Background(), jsCode)
	require.NoError(t, err)

	// Test with zero timeout (should use default)
	opts := Options{Timeout: 0}

	result, err := jsFunc.Run(context.Background(), map[string]any{"value": 5}, map[string]any{"value": 3}, opts)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"result": int64(8)}, result)
}

func TestJSFunction_Run_Isolation(t *testing.T) {
	// Test that each run uses a fresh VM instance
	jsCode := "function(a, b) { if (!a.counter) a.counter = 0; a.counter++; return { counter: a.counter, sum: a.value + b.value }; }"

	jsFunc, err := New(context.Background(), jsCode)
	require.NoError(t, err)

	opts := Options{Timeout: 100 * time.Millisecond}

	// First run
	firstSession := map[string]any{}
	result1, err := jsFunc.Run(context.Background(), firstSession, map[string]any{"value": 3}, opts)
	require.NoError(t, err)

	assert.Equal(t, int64(1), result1["counter"])

	// Second run - should start fresh
	secondSession := map[string]any{}
	result2, err := jsFunc.Run(context.Background(), secondSession, map[string]any{"value": 20}, opts)
	require.NoError(t, err)

	assert.Equal(t, int64(1), result2["counter"]) // Should be 1 again, not 2
}

func TestJSFunction_Run_ConsoleLog(t *testing.T) {
	// Test that console.log works and doesn't cause panics
	jsCode := "function(a, b) { console.log('Session:', a, 'Input:', b); return { result: a.value + b.value }; }"

	jsFunc, err := New(context.Background(), jsCode)
	require.NoError(t, err)

	opts := Options{Timeout: 100 * time.Millisecond}

	// This should not panic and should execute successfully
	result, err := jsFunc.Run(context.Background(), map[string]any{"value": 5}, map[string]any{"value": 3}, opts)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"result": int64(8)}, result)
}

func TestJSFunction_Run_ConsoleLog_WithBuffer(t *testing.T) {
	// Test that console.log actually produces log output
	jsCode := "function(a, b) { console.log('Test message:', a, b); console.error('Error message:', a.value + b.value); return { result: a.value + b.value }; }"

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create context with logger
	ctx := logger.WithContext(context.Background())

	jsFunc, err := New(ctx, jsCode)
	require.NoError(t, err)

	opts := Options{Timeout: 100 * time.Millisecond}

	// Run the function
	result, err := jsFunc.Run(ctx, map[string]any{"value": 5}, map[string]any{"value": 3}, opts)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"result": int64(8)}, result)

	// Verify that console.log and console.error messages are in the output
	outputStr := buf.String()
	assert.Contains(t, outputStr, "Test message:")
	assert.Contains(t, outputStr, "Error message:")
	assert.Contains(t, outputStr, "5")
	assert.Contains(t, outputStr, "3")
	assert.Contains(t, outputStr, "8")
}

func TestJSFunction_Run_ConsoleLog_MultipleCalls_WithBuffer(t *testing.T) {
	// Test multiple console.log calls in the same function
	jsCode := "function(a, b) { console.log('First:', a); console.log('Second:', b); console.log('Sum:', a.value + b.value); return { result: a.value + b.value }; }"

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create context with logger
	ctx := logger.WithContext(context.Background())

	jsFunc, err := New(ctx, jsCode)
	require.NoError(t, err)

	opts := Options{Timeout: 100 * time.Millisecond}

	// Run the function
	result, err := jsFunc.Run(ctx, map[string]any{"value": 10}, map[string]any{"value": 20}, opts)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"result": int64(30)}, result)

	// Verify all console.log messages are present
	outputStr := buf.String()
	assert.Contains(t, outputStr, "First:")
	assert.Contains(t, outputStr, "Second:")
	assert.Contains(t, outputStr, "Sum:")
	assert.Contains(t, outputStr, "10")
	assert.Contains(t, outputStr, "20")
	assert.Contains(t, outputStr, "30")
}

func TestJSFunction_Run_ConsoleLog_ComplexObjects_WithBuffer(t *testing.T) {
	// Test console.log with complex objects
	jsCode := "function(session, input) { console.log('Session object:', session); console.log('Input object:', input); return { sum: session.value + input.value }; }"

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create context with logger
	ctx := logger.WithContext(context.Background())

	jsFunc, err := New(ctx, jsCode)
	require.NoError(t, err)

	opts := Options{Timeout: 100 * time.Millisecond}

	// Run the function
	result, err := jsFunc.Run(ctx, map[string]any{"value": 10}, map[string]any{"value": 20}, opts)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"sum": int64(30)}, result)

	// Verify console.log messages are present
	outputStr := buf.String()
	assert.Contains(t, outputStr, "Session object:")
	assert.Contains(t, outputStr, "Input object:")
	assert.Contains(t, outputStr, "value")
	assert.Contains(t, outputStr, "10")
	assert.Contains(t, outputStr, "20")
}

func TestJSFunction_Run_ConsoleLog_Error_WithBuffer(t *testing.T) {
	// Test that console.error works and produces error level logs
	jsCode := "function(a, b) { console.error('Error message:', a.value + b.value); return { result: a.value + b.value }; }"

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create context with logger
	ctx := logger.WithContext(context.Background())

	jsFunc, err := New(ctx, jsCode)
	require.NoError(t, err)

	opts := Options{Timeout: 100 * time.Millisecond}

	// Run the function
	result, err := jsFunc.Run(ctx, map[string]any{"value": 5}, map[string]any{"value": 3}, opts)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"result": int64(8)}, result)

	// Verify that console.error message is in the output
	outputStr := buf.String()
	assert.Contains(t, outputStr, "Error message:")
	assert.Contains(t, outputStr, "8")
}

func TestJSFunction_Run_LargeData(t *testing.T) {
	// Test with large JSON objects
	largeSession := make(map[string]any)
	largeInput := make(map[string]any)

	for i := 0; i < 1000; i++ {
		largeSession[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		largeInput[fmt.Sprintf("inputKey%d", i)] = i
	}

	jsCode := "function(session, input) { return { sessionKeys: Object.keys(session).length, inputKeys: Object.keys(input).length, sum: Object.keys(session).length + Object.keys(input).length }; }"

	jsFunc, err := New(context.Background(), jsCode)
	require.NoError(t, err)

	opts := Options{Timeout: 1 * time.Second}

	result, err := jsFunc.Run(context.Background(), largeSession, largeInput, opts)
	assert.NoError(t, err)

	assert.Equal(t, int64(1000), result["sessionKeys"])
	assert.Equal(t, int64(1000), result["inputKeys"])
	assert.Equal(t, int64(2000), result["sum"])
}

func TestJSFunction_Run_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		jsCode      string
		sessionArgs map[string]any
		inputArgs   map[string]any
		expected    map[string]any
	}{
		{
			name:        "empty arrays",
			jsCode:      "function(a, b) { return { lengths: [a.value.length, b.value.length] }; }",
			sessionArgs: map[string]any{"value": []any{}},
			inputArgs:   map[string]any{"value": []any{}},
			expected:    map[string]any{"lengths": []any{int64(0), int64(0)}},
		},
		{
			name:        "null values",
			jsCode:      "function(a, b) { return { aNull: a === null, bNull: b === null, aType: typeof a, bType: typeof b }; }",
			sessionArgs: nil,
			inputArgs:   nil,
			expected:    map[string]any{"aNull": true, "aType": "object", "bNull": true, "bType": "object"},
		},
		{
			name:        "boolean values",
			jsCode:      "function(a, b) { return { aBool: typeof a.value, bBool: typeof b.value, result: a.value && b.value }; }",
			sessionArgs: map[string]any{"value": true},
			inputArgs:   map[string]any{"value": false},
			expected:    map[string]any{"aBool": "boolean", "bBool": "boolean", "result": false},
		},
		{
			name:        "string values",
			jsCode:      "function(a, b) { return { concatenated: a.value + ' ' + b.value }; }",
			sessionArgs: map[string]any{"value": "hello"},
			inputArgs:   map[string]any{"value": "world"},
			expected:    map[string]any{"concatenated": "hello world"},
		},
		{
			name:        "number values",
			jsCode:      "function(a, b) { return { sum: a.value + b.value, product: a.value * b.value, quotient: a.value / b.value }; }",
			sessionArgs: map[string]any{"value": 10},
			inputArgs:   map[string]any{"value": 5},
			expected:    map[string]any{"product": int64(50), "quotient": int64(2), "sum": int64(15)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsFunc, err := New(context.Background(), tt.jsCode)
			require.NoError(t, err)

			opts := Options{Timeout: 100 * time.Millisecond}

			result, err := jsFunc.Run(context.Background(), tt.sessionArgs, tt.inputArgs, opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSFunction_Run_PanicRecovery(t *testing.T) {
	// Test that panics in the JavaScript code are properly recovered
	jsCode := "function(a, b) { throw new Error('Test error'); }"

	jsFunc, err := New(context.Background(), jsCode)
	require.NoError(t, err)

	opts := Options{Timeout: 100 * time.Millisecond}

	result, err := jsFunc.Run(context.Background(), map[string]any{"value": 5}, map[string]any{"value": 3}, opts)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrJSRuntimeError)
	assert.Contains(t, err.Error(), "Test error")
}

func BenchmarkJSFunction_Run(b *testing.B) {
	jsCode := "function(session, input) { return { result: session.value + input.value, timestamp: Date.now() }; }"
	jsFunc, err := New(context.Background(), jsCode)
	require.NoError(b, err)

	sessionArgs := map[string]any{"value": 10, "id": "session1"}
	inputArgs := map[string]any{"value": 20, "id": "input1"}
	opts := Options{Timeout: 100 * time.Millisecond}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := jsFunc.Run(context.Background(), sessionArgs, inputArgs, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSFunction_New(b *testing.B) {
	jsCode := "function(session, input) { return session.value + input.value; }"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New(context.Background(), jsCode)
		if err != nil {
			b.Fatal(err)
		}
	}
}
