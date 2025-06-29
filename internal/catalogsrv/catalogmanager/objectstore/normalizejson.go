package objectstore

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/anand-gl/jsoncanonicalizer"
)

// NormalizeJSON sorts the keys of the JSON and returns the canonical representation
func NormalizeJSON(data []byte) ([]byte, error) {
	return jsoncanonicalizer.Transform(data)

	// We don't use the custom sorting function here to use a more standardized approach provided by the jsoncanonicalizer package.
	/*
		var jsonObj interface{}

		// Unmarshal the JSON data into a generic map
		err := json.Unmarshal(data, &jsonObj)
		if err != nil {
			return nil, err
		}

		// Marshal back the object into sorted JSON
		normalizedJSON, err := marshalSorted(jsonObj)
		if err != nil {
			return nil, err
		}

		return normalizedJSON, nil
	*/
}

var _ = marshalSorted // This is to ensure that the function is included in the package for testing purposes

// marshalSorted recursively sorts keys and marshals JSON in a deterministic way
func marshalSorted(obj interface{}) ([]byte, error) {
	switch v := obj.(type) {
	case map[string]interface{}:
		// Create a buffer to hold the sorted JSON output
		buf := &bytes.Buffer{}
		buf.WriteByte('{')

		// Sort the keys
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Write the keys and values in sorted order
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}

			// Marshal the key as a string
			keyData, err := json.Marshal(k)
			if err != nil {
				return nil, err
			}
			buf.Write(keyData)
			buf.WriteByte(':')

			// Marshal the value (which could be a complex structure)
			valueData, err := marshalSorted(v[k])
			if err != nil {
				return nil, err
			}
			buf.Write(valueData)
		}

		buf.WriteByte('}')
		return buf.Bytes(), nil

	case []interface{}:
		// For arrays, process each element
		buf := &bytes.Buffer{}
		buf.WriteByte('[')
		for i, elem := range v {
			if i > 0 {
				buf.WriteByte(',')
			}

			marshaledElem, err := marshalSorted(elem)
			if err != nil {
				return nil, err
			}
			buf.Write(marshaledElem)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil

	default:
		// For primitive types, just marshal normally
		return json.Marshal(v)
	}
}
