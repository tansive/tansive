package types

import "encoding/json"

type NullableString struct {
	Value string
	Valid bool // Valid is true if Value is not nil
}

func (ns NullableString) String() string {
	if ns.Valid { // not needed, but for clarity
		return ns.Value
	}
	return ""
}

func (ns NullableString) IsNil() bool {
	return !ns.Valid || ns.Value == ""
}

func (ns *NullableString) Set(value string) {
	ns.Value = value
	ns.Valid = true
}

// implement json.Marshaler interface
func (ns NullableString) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.Value)
	}
	return json.Marshal(nil)
}

func (ns *NullableString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		ns.Value = ""
		ns.Valid = false
		return nil
	}
	ns.Valid = true
	return json.Unmarshal(data, &ns.Value)
}

func NullableStringFrom(s string) NullableString {
	return NullableString{Value: s, Valid: true}
}

func NullString() NullableString {
	return NullableString{Value: "", Valid: false}
}

var _ json.Marshaler = &NullableString{}   // Ensure NullableString implements json.Marshaler
var _ json.Unmarshaler = &NullableString{} // Ensure NullableString implements json.Unmarshaler
var _ Nullable = &NullableString{}         // Ensure NullableString implements Nullable interface
