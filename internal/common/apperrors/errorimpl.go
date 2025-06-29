package apperrors

import (
	"errors"
	"strings"
)

// appError implements the apperrors.Error interface. It provides a concrete implementation
// of the Error interface with support for error wrapping, status codes, and message formatting.
type appError struct {
	msg           string  // primary error message
	base          error   // base error for errors.Is/As compatibility
	wrappedErrors []error // additional wrapped errors
	statuscode    int     // HTTP status code
	expandError   bool    // controls error message expansion
	prefix        string  // optional message prefix
	suffix        string  // optional message suffix
}

// Error returns the formatted error message without mutating state.
// The message includes prefix and suffix if set.
func (e *appError) Error() string {
	msg := e.msg
	if e.prefix != "" {
		msg = e.prefix + ": " + msg
	}
	if e.suffix != "" {
		msg = msg + ": " + e.suffix
	}
	return msg
}

// ErrorAll returns the full message including wrapped errors if expandError is true.
// Otherwise, returns the same as Error().
func (e *appError) ErrorAll() string {
	if !e.expandError {
		return e.Error()
	}
	var b strings.Builder
	b.WriteString(e.Error())
	for _, err := range e.wrappedErrors {
		b.WriteString("; ")
		b.WriteString(err.Error())
	}
	return b.String()
}

// Unwrap returns the base error for compatibility with errors.Is / errors.As.
func (e *appError) Unwrap() error {
	return e.base
}

// UnwrapAll returns all wrapped errors in the order they were added.
func (e *appError) UnwrapAll() []error {
	return e.wrappedErrors
}

// Msg creates a new error with a new message and wraps the original error.
// The new error inherits the status code from the original.
func (e *appError) Msg(msg string) Error {
	return &appError{
		msg:           msg,
		base:          e,
		wrappedErrors: append([]error{e}, e.wrappedErrors...),
		statuscode:    e.statuscode,
	}
}

// New creates a fresh error using the current error as a template.
// The new error inherits the status code but starts with a new message.
func (e *appError) New(msg string) Error {
	return &appError{
		msg:        msg,
		base:       e,
		statuscode: e.statuscode,
	}
}

// MsgErr creates a new error with a message and wraps additional errors.
// The new error inherits the status code from the original.
func (e *appError) MsgErr(msg string, errs ...error) Error {
	all := append([]error{e}, errs...)
	return &appError{
		msg:           msg,
		base:          e,
		wrappedErrors: all,
		statuscode:    e.statuscode,
	}
}

// Err creates a new error by attaching additional errors to the current error.
// The new error maintains the original message and status code.
func (e *appError) Err(errs ...error) Error {
	all := append([]error{e}, errs...)
	return &appError{
		msg:           e.msg,
		base:          e,
		wrappedErrors: all,
		statuscode:    e.statuscode,
	}
}

// Prefix returns a shallow copy with an updated prefix.
// The original error remains unchanged.
func (e *appError) Prefix(p string) Error {
	cp := *e
	cp.prefix = p
	return &cp
}

// Suffix returns a shallow copy with an updated suffix.
// The original error remains unchanged.
func (e *appError) Suffix(s string) Error {
	cp := *e
	cp.suffix = s
	return &cp
}

// SetExpandError returns a shallow copy with an updated expansion flag.
// The original error remains unchanged.
func (e *appError) SetExpandError(flag bool) Error {
	cp := *e
	cp.expandError = flag
	return &cp
}

// SetStatusCode returns a shallow copy with an updated status code.
// The original error remains unchanged.
func (e *appError) SetStatusCode(code int) Error {
	cp := *e
	cp.statuscode = code
	return &cp
}

// StatusCode returns the current HTTP status code.
func (e *appError) StatusCode() int {
	return e.statuscode
}

// New creates a root-level appError with the given message.
// This is the entry point for creating new errors.
func New(msg string) Error {
	return &appError{
		msg: msg,
	}
}

// Is checks if the error is equal to the target error by checking
// both the base error and all wrapped errors.
func (e *appError) Is(target error) bool {
	if target == nil {
		return false
	}
	if errors.Is(e.base, target) {
		return true
	}
	for _, err := range e.wrappedErrors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
