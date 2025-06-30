// Package types provides nullable type implementations for handling optional values.
package types

// Nullable defines the interface for types that can represent null/nil values.
// Types implementing this interface can distinguish between a zero value and a null value,
// which is useful for database operations and JSON serialization where null has semantic meaning.
type Nullable interface {
	// IsNil returns true if the value is null/nil, false otherwise.
	// This allows distinguishing between a zero value and an explicitly null value.
	IsNil() bool
}
