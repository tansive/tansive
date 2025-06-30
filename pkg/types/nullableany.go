// Package types provides nullable type implementations for handling optional values.
package types

import (
	"bytes"
	"encoding/json"
	"errors"
)

// NullableAny represents a nullable value that can hold any JSON-serializable type.
// It provides type-safe handling of optional values with proper JSON marshaling/unmarshaling.
// The value is stored internally as JSON to maintain type fidelity across serialization.
type NullableAny struct {
	value json.RawMessage
	valid bool // Valid is true if Value is not nil
}

// IsNil returns true if the NullableAny is null/nil, false otherwise.
// This implements the Nullable interface.
func (ns NullableAny) IsNil() bool {
	return !ns.valid
}

// Set assigns a value to the NullableAny, converting it to JSON format.
// Accepts any JSON-serializable type and validates the resulting JSON.
// Returns an error if the value cannot be marshaled to JSON.
func (ns *NullableAny) Set(value any) error {
	var jsonValue json.RawMessage

	switch v := value.(type) {
	case json.RawMessage:
		// If already a json.RawMessage, validate it
		if !json.Valid(v) {
			ns.value = nil
			ns.valid = false
			return errors.New("value is not valid JSON")
		}
		jsonValue = v
	case []byte:
		// Check if []byte contains valid JSON
		if !json.Valid(v) {
			// If not valid JSON, try marshaling it
			marshaledValue, err := json.Marshal(value)
			if err != nil {
				ns.value = nil
				ns.valid = false
				return err
			}
			jsonValue = marshaledValue
		} else {
			jsonValue = v
		}
	default:
		// Marshal any other type
		marshaledValue, err := json.Marshal(value)
		if err != nil {
			ns.value = nil
			ns.valid = false
			return err
		}
		jsonValue = marshaledValue
	}

	// Assign validated/marshaled value
	ns.value = jsonValue
	ns.valid = true
	return nil
}

// Get returns the underlying value as interface{}.
// Returns nil if the NullableAny is null/nil.
// The value is unmarshaled from the internal JSON representation.
func (ns NullableAny) Get() any {
	if ns.valid {
		var v any
		err := json.Unmarshal(ns.value, &v)
		if err != nil {
			return nil
		}
		return v
	}
	return nil
}

// Equals compares two NullableAny values for equality.
// Returns true if both values are null or if both have the same JSON representation.
func (ns NullableAny) Equals(value NullableAny) bool {
	if ns.valid && value.valid {
		return bytes.Equal(ns.value, value.value)
	}
	return ns.valid == value.valid
}

// GetAs unmarshals the value into the provided target interface.
// Returns an error if the NullableAny is null/nil or if unmarshaling fails.
// The target must be a pointer to the desired type.
func (ns NullableAny) GetAs(v any) error {
	if ns.valid {
		return json.Unmarshal(ns.value, v)
	}
	return errors.New("value is not set")
}

// MarshalJSON implements the json.Marshaler interface.
// Returns the raw JSON value if valid, or null if the value is nil.
func (ns NullableAny) MarshalJSON() ([]byte, error) {
	if ns.valid {
		return ns.value, nil
	}
	return json.Marshal(nil)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Accepts JSON data and sets the NullableAny value, validating the JSON format.
// Returns an error if the JSON is invalid.
func (ns *NullableAny) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		ns.value = nil
		ns.valid = false
		return nil
	}
	if !json.Valid(data) {
		ns.value = nil
		ns.valid = false
		return errors.New("invalid JSON")
	}
	ns.value = data
	ns.valid = true
	return nil
}

// NullableAnyFrom creates a new NullableAny from any value.
// Returns an error if the value cannot be marshaled to JSON.
// This is a convenience function for creating NullableAny instances.
func NullableAnyFrom(value any) (NullableAny, error) {
	var na NullableAny
	err := na.Set(value)
	if err != nil {
		return NullableAny{}, err
	}
	return na, nil
}

// NullableAnySetRaw creates a NullableAny directly from raw JSON data.
// This bypasses the marshaling step and is useful when you already have valid JSON.
// The caller is responsible for ensuring the JSON is valid.
func NullableAnySetRaw(value json.RawMessage) NullableAny {
	return NullableAny{
		value: value,
		valid: true,
	}
}

// NilAny creates a new NullableAny that represents a null value.
// This is equivalent to a nil NullableAny but provides a clear constructor.
func NilAny() NullableAny {
	// Return a NullableAny that is nil
	return NullableAny{
		value: nil,
		valid: false,
	}
}

var _ json.Marshaler = &NullableAny{}
var _ json.Unmarshaler = &NullableAny{}
var _ Nullable = &NullableAny{}
var _ json.Marshaler = NullableAny{}
