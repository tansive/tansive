package httpx

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// ResponseWriter is a wrapper around http.ResponseWriter that tracks if headers were written
// and provides additional functionality for response handling.
type ResponseWriter struct {
	http.ResponseWriter
	written bool
	status  int
}

// NewResponseWriter creates a new ResponseWriter wrapping the provided http.ResponseWriter.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{ResponseWriter: w}
}

// WriteHeader implements http.ResponseWriter.WriteHeader.
// If headers were already written, this is a no-op.
func (rw *ResponseWriter) WriteHeader(code int) {
	if rw.written {
		return
	}
	rw.status = code
	rw.written = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write implements http.ResponseWriter.Write.
// If headers were not written, writes StatusOK (200) header first.
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Written reports whether headers or body were written.
func (rw *ResponseWriter) Written() bool {
	return rw.written
}

// Status returns the status code. Returns http.StatusOK (200) if not set.
func (rw *ResponseWriter) Status() int {
	if rw.status == 0 {
		return http.StatusOK
	}
	return rw.status
}

// Flush implements http.Flusher if the underlying writer supports it.
func (rw *ResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker if the underlying writer supports it.
func (rw *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrHijacked
	}
	return hj.Hijack()
}

// Push implements http.Pusher if supported (for HTTP/2).
func (rw *ResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// ReadFrom implements io.ReaderFrom if supported.
// Falls back to io.Copy if not supported.
func (rw *ResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	if rf, ok := rw.ResponseWriter.(io.ReaderFrom); ok {
		if !rw.written {
			rw.WriteHeader(http.StatusOK)
		}
		return rf.ReadFrom(r)
	}
	return io.Copy(rw, r)
}
