package stdiorunner

import (
	"io"

	"github.com/tansive/tansive/internal/tangent/tangentcommon"
)

type WriterType string

const (
	StdoutWriter WriterType = "stdout"
	StderrWriter WriterType = "stderr"
)

type writer struct {
	writerType WriterType
	writers    []*tangentcommon.IOWriters
}

// Write writes the byte slice p to the appropriate stream (Out or Err) of each IOWriters target,
// depending on the writerType (StdoutWriter or StderrWriter).
//
// It returns the number of bytes successfully written and an error if any writer failed or performed
// a partial write. If at least one writer short-writes, io.ErrShortWrite is returned.
// If no writers write anything, the first error encountered (if any) is returned.
func (w *writer) Write(p []byte) (int, error) {
	var (
		minWritten int = len(p)
		wrote      bool
		anyShort   bool
		firstErr   error
	)

	if len(w.writers) == 0 {
		return len(p), nil
	}

	for _, wtr := range w.writers {
		var target io.Writer
		switch w.writerType {
		case StdoutWriter:
			target = wtr.Out
		case StderrWriter:
			target = wtr.Err
		default:
			continue
		}

		if target == nil {
			continue
		}

		nn, err := target.Write(p)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		if nn > 0 {
			wrote = true
			if nn < len(p) {
				anyShort = true
			}
			if nn < minWritten {
				minWritten = nn
			}
		}
	}

	if !wrote && firstErr != nil {
		return 0, firstErr
	}
	if anyShort {
		return minWritten, io.ErrShortWrite
	}
	if wrote {
		return len(p), firstErr
	}
	return 0, nil
}

// NewWriter constructs an io.Writer that delegates to the Out or Err streams of each IOWriters.
func NewWriter(writerType WriterType, writers ...*tangentcommon.IOWriters) io.Writer {
	return &writer{
		writerType: writerType,
		writers:    writers,
	}
}
