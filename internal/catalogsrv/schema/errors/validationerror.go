// Package schemas provides data structures and functions for handling errors in the API.
package errors

import (
	"bytes"
	"strings"
)

// ValidationError represents an error that occurs during validation.
type ValidationError struct {
	Field  string // The field that caused the validation error.
	Value  any    // The value that caused the validation error.
	ErrStr string // The error message.
}

// Error allows ValidationError to satisfy the error interface.
func (ve ValidationError) Error() string {
	if len(ve.Field) > 0 {
		return ve.Field + ": " + ve.ErrStr
	} else {
		return ve.ErrStr
	}
}

// ErrInvalidSchema is an error indicating that the schema is invalid.
var ErrInvalidSchema = ValidationError{
	Field:  "invalid input",
	Value:  "",
	ErrStr: "unable to parse schema",
}

// ValidationErrors represents a collection of validation errors.
type ValidationErrors []ValidationError

// Error allows ValidationErrors to satisfy the error interface.
func (ves ValidationErrors) Error() string {
	buff := bytes.NewBufferString("")

	for i := 0; i < len(ves); i++ {
		buff.WriteString(ves[i].Error())
		buff.WriteString("; ")
	}

	return strings.TrimSpace(buff.String())
}

// inQuotes returns the string s surrounded by double quotes.
func InQuotes(s string) string {
	return "'" + s + "'"
}
