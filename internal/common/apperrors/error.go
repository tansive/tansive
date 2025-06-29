// Package apperrors provides a flexible error handling system that supports error wrapping,
// status codes, and message formatting. It implements the standard error interface while
// adding extended functionality for error chaining, status code management, and message
// customization.
package apperrors

// Error defines the interface for application errors. It extends the standard error
// interface with additional methods for error wrapping, message manipulation, and
// status code management. All methods return Error to support method chaining.
type Error interface {
	error
	Unwrap() error // support for errors.Is / errors.As

	// Extended methods
	New(msg string) Error                  // creates a new error using current as template
	Msg(msg string) Error                  // creates a new error with message and wraps original
	MsgErr(msg string, err ...error) Error // creates error with message and wraps extra errors
	Err(err ...error) Error                // attaches additional errors to current error
	SetExpandError(bool) Error             // controls whether ErrorAll expands wrapped errors
	SetStatusCode(int) Error               // sets HTTP status code for the error
	StatusCode() int                       // returns the current status code
	Prefix(string) Error                   // adds a prefix to the error message
	Suffix(string) Error                   // adds a suffix to the error message
	ErrorAll() string                      // returns full message including wrapped errors
	UnwrapAll() []error                    // returns all wrapped errors
}
