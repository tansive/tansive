// Package types provides nullable type implementations for handling optional values.
package types

import "encoding/json"

// NullableString represents a nullable string value.
// It can distinguish between an empty string and a null value,
// which is useful for database operations and JSON serialization.
type NullableString struct {
	Value string
	Valid bool // Valid is true if Value is not nil
}

// String returns the string value if valid, or an empty string if null.
// This implements the fmt.Stringer interface for convenient string conversion.
func (ns NullableString) String() string {
	if ns.Valid { // not needed, but for clarity
		return ns.Value
	}
	return ""
}

// IsNil returns true if the NullableString is null/nil, false otherwise.
// This implements the Nullable interface.
// Note: An empty string with Valid=true is not considered nil.
func (ns NullableString) IsNil() bool {
	return !ns.Valid || ns.Value == ""
}

// Set assigns a string value to the NullableString and marks it as valid.
// This method allows changing the value of an existing NullableString.
func (ns *NullableString) Set(value string) {
	ns.Value = value
	ns.Valid = true
}

// MarshalJSON implements the json.Marshaler interface.
// Returns the string value as JSON if valid, or null if the value is nil.
func (ns NullableString) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.Value)
	}
	return json.Marshal(nil)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Accepts JSON data and sets the NullableString value.
// Handles null values by setting Valid to false and Value to empty string.
func (ns *NullableString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		ns.Value = ""
		ns.Valid = false
		return nil
	}
	ns.Valid = true
	return json.Unmarshal(data, &ns.Value)
}

// NullableStringFrom creates a new NullableString from a string value.
// The resulting NullableString will be valid and contain the provided string.
// This is a convenience function for creating valid NullableString instances.
func NullableStringFrom(s string) NullableString {
	return NullableString{Value: s, Valid: true}
}

// NullString creates a new NullableString that represents a null value.
// The resulting NullableString will have Valid=false and an empty Value.
// This is a convenience function for creating null NullableString instances.
func NullString() NullableString {
	return NullableString{Value: "", Valid: false}
}

var _ json.Marshaler = &NullableString{}   // Ensure NullableString implements json.Marshaler
var _ json.Unmarshaler = &NullableString{} // Ensure NullableString implements json.Unmarshaler
var _ Nullable = &NullableString{}         // Ensure NullableString implements Nullable interface
