package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// AsciinemaEvent represents a single event in an asciinema recording.
// Contains timing information, stream type, and event data.
type AsciinemaEvent struct {
	Time   float64 `json:"time"`   // elapsed time in seconds since recording start
	Stream string  `json:"stream"` // stream type: "o" for output, "i" for input
	Data   string  `json:"data"`   // event data content
}

// AsciinemaWriter provides functionality to create and write asciinema recordings.
// Supports writing events with timestamps and proper asciinema v2 format.
type AsciinemaWriter struct {
	file   *os.File
	writer *bufio.Writer
	start  time.Time
	closed bool
}

// NewAsciinemaWriter creates a new asciinema file and writes the header.
// Returns the writer instance and any error encountered during creation.
func NewAsciinemaWriter(filename string) (*AsciinemaWriter, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	w := bufio.NewWriter(f)

	// Write the Asciinema v2 header
	header := map[string]any{
		"version":   2,
		"width":     80,
		"height":    24,
		"timestamp": time.Now().Unix(),
		"env": map[string]string{
			"SHELL": os.Getenv("SHELL"),
			"TERM":  "xterm-256color",
		},
	}
	headerBytes, _ := json.Marshal(header)
	fmt.Fprintln(w, string(headerBytes))

	return &AsciinemaWriter{
		file:   f,
		writer: w,
		start:  time.Now(),
	}, nil
}

// Write appends a new event (either input or output) with a timestamp.
// Returns an error if the writer is closed or write operation fails.
func (a *AsciinemaWriter) Write(stream string, data string) error {
	if a.closed {
		return fmt.Errorf("asciinema writer already closed")
	}
	elapsed := time.Since(a.start).Seconds()
	event := []any{elapsed, stream, data}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.writer, string(eventBytes))
	return err
}

// Close flushes and closes the underlying file.
// Ensures all buffered data is written before closing.
func (a *AsciinemaWriter) Close() error {
	if a.closed {
		return nil
	}
	a.closed = true
	if err := a.writer.Flush(); err != nil {
		return err
	}
	return a.file.Close()
}
