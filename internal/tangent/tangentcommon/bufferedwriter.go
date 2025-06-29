package tangentcommon

import (
	"bytes"
)

// BufferedWriter is a simple buffer that accumulates all writes in memory.
// Implements io.Writer and io.StringWriter interfaces for flexible I/O operations.
type BufferedWriter struct {
	buf bytes.Buffer // underlying buffer for data storage
}

// Write implements io.Writer interface.
// Appends the given bytes to the buffer and returns the number of bytes written.
func (b *BufferedWriter) Write(p []byte) (int, error) {
	return b.buf.Write(p)
}

// WriteString implements io.StringWriter interface.
// Appends the given string to the buffer and returns the number of bytes written.
func (b *BufferedWriter) WriteString(s string) (int, error) {
	return b.buf.WriteString(s)
}

// String returns the accumulated contents as a string.
// Returns the complete buffer contents as a string representation.
func (b *BufferedWriter) String() string {
	return b.buf.String()
}

// Bytes returns the accumulated contents as a byte slice.
// Returns a copy of the buffer contents as a byte array.
func (b *BufferedWriter) Bytes() []byte {
	return b.buf.Bytes()
}

// Reset clears the buffer.
// Removes all accumulated data and resets the buffer to empty state.
func (b *BufferedWriter) Reset() {
	b.buf.Reset()
}

// Len returns the number of bytes in the buffer.
// Returns the current size of the accumulated data.
func (b *BufferedWriter) Len() int {
	return b.buf.Len()
}

// NewBufferedWriter constructs a new BufferedWriter.
// Returns an initialized buffer ready for writing operations.
func NewBufferedWriter() *BufferedWriter {
	return &BufferedWriter{}
}
