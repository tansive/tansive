package objectstore

import (
	"reflect"

	"encoding/json"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/apperrors"
)

type ObjectStorageRepresentation struct {
	Version     string                      `json:"version"`
	Type        catcommon.CatalogObjectType `json:"type"`
	Description string                      `json:"description"`
	Spec        json.RawMessage             `json:"spec"`
	Values      json.RawMessage             `json:"values"`
	Reserved    json.RawMessage             `json:"reserved"`
	Entropy     []byte                      `json:"entropy"`
}

// Serialize converts the SchemaStorageRepresentation to a JSON byte array
func (s *ObjectStorageRepresentation) Serialize() ([]byte, apperrors.Error) {
	j, err := json.Marshal(s)
	if err != nil {
		return nil, apperrors.New("failed to serialize object storage representation")
	}
	return j, nil
}

func (s *ObjectStorageRepresentation) SetEntropy(entropy []byte) {
	if entropy == nil {
		s.Entropy = nil
		return
	}
	s.Entropy = entropy
}

// GetHash returns the SHA-512 hash of the normalized SchemaStorageRepresentation
func (s *ObjectStorageRepresentation) GetHash() string {
	sz, err := s.Serialize()
	if err != nil {
		return ""
	}
	// Normalize the JSON, so 2 equivalent representations yield the same hash
	nsz, e := NormalizeJSON(sz)
	if e != nil {
		return ""
	}
	hash := HexEncodedSHA512(nsz)
	return hash
}

// Size returns the approximate size of the SchemaStorageRepresentation in bytes
func (s *ObjectStorageRepresentation) Size() int {
	return len(s.Spec) + len(s.Version) + len(s.Type)
}

func (s *ObjectStorageRepresentation) DiffersInSpec(other *ObjectStorageRepresentation) bool {
	if other == nil {
		return true
	}
	// Compare the Schema field only for differences do a byte compare
	res, err := jsonEqual(s.Spec, other.Spec)
	return err != nil || !res
}

func jsonEqual(a, b json.RawMessage) (bool, error) {
	var objA any
	var objB any

	if err := json.Unmarshal([]byte(a), &objA); err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(b), &objB); err != nil {
		return false, err
	}

	return reflect.DeepEqual(objA, objB), nil
}
