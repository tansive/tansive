package tangentcommon

import (
	"testing"
)

func TestBufferedWriter(t *testing.T) {
	// Test NewBufferedWriter
	writer := NewBufferedWriter()
	if writer == nil {
		t.Fatal("NewBufferedWriter returned nil")
	}

	// Test Write
	testData := []byte("Hello, World!")
	n, err := writer.Write(testData)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write returned %d bytes, want %d", n, len(testData))
	}

	// Test WriteString
	testString := " Testing!"
	n, err = writer.WriteString(testString)
	if err != nil {
		t.Errorf("WriteString failed: %v", err)
	}
	if n != len(testString) {
		t.Errorf("WriteString returned %d bytes, want %d", n, len(testString))
	}

	// Test String
	expected := "Hello, World! Testing!"
	if got := writer.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}

	// Test Bytes
	expectedBytes := []byte(expected)
	gotBytes := writer.Bytes()
	if len(gotBytes) != len(expectedBytes) {
		t.Errorf("Bytes() length = %d, want %d", len(gotBytes), len(expectedBytes))
	}
	for i := range expectedBytes {
		if gotBytes[i] != expectedBytes[i] {
			t.Errorf("Bytes()[%d] = %v, want %v", i, gotBytes[i], expectedBytes[i])
		}
	}

	// Test Reset
	writer.Reset()
	if got := writer.String(); got != "" {
		t.Errorf("After Reset, String() = %q, want empty string", got)
	}
}

func TestBufferedWriterEmpty(t *testing.T) {
	writer := NewBufferedWriter()

	// Test empty buffer
	if got := writer.String(); got != "" {
		t.Errorf("Empty buffer String() = %q, want empty string", got)
	}

	if got := writer.Bytes(); len(got) != 0 {
		t.Errorf("Empty buffer Bytes() length = %d, want 0", len(got))
	}
}
