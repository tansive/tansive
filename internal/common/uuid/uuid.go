package uuid

import (
	"encoding/binary"
	"time"

	"github.com/google/uuid"
)

// Package uuid provides UUID functionality with a focus on UUIDv7 (time-ordered UUIDs).
// It wraps github.com/google/uuid package and sets version 7 as the default.

// UUID represents a UUID, aliased from github.com/google/uuid.UUID
type UUID = uuid.UUID

// UUID7 generates a new UUIDv7. Panics if UUID generation fails.
func UUID7() UUID {
	uuidv7, _ := uuid.NewV7()
	return uuidv7
}

// NewRandom returns a new random UUIDv7 and any error encountered during generation.
func NewRandom() (UUID, error) {
	return uuid.NewV7()
}

// New returns a new random UUIDv7. Panics if UUID generation fails.
func New() UUID {
	uuidv7, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return uuidv7
}

// Parse parses a UUID string into a UUID value. Returns an error if the string is not a valid UUID.
func Parse(s string) (UUID, error) {
	return uuid.Parse(s)
}

// MustParse parses a UUID string and panics if the string is not a valid UUID.
func MustParse(s string) UUID {
	return uuid.MustParse(s)
}

// IsUUIDv7 reports whether the given UUID is a valid UUIDv7.
func IsUUIDv7(id UUID) bool {
	return id.Version() == uuid.Version(7)
}

// GetTimestampFromUUID extracts the timestamp from a UUIDv7 and returns it as a time.Time.
// The timestamp is extracted from the top 48 bits of the UUID.
func GetTimestampFromUUID(u UUID) time.Time {
	tsMillis := binary.BigEndian.Uint64(u[0:8]) >> 16
	// Check if the value is within int64 bounds
	if tsMillis > uint64(1<<63-1) {
		// If overflow would occur, return the maximum valid time
		return time.UnixMilli(1<<63 - 1)
	}
	return time.UnixMilli(int64(tsMillis))
}

// CompareUUIDv7 compares two UUIDv7 values.
// Returns:
//   - -1 if a was created before b
//   - 0 if a == b
//   - +1 if a was created after b
func CompareUUIDv7(a, b UUID) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// IsBefore reports whether a was created before b.
func IsBefore(a, b UUID) bool {
	return CompareUUIDv7(a, b) == -1
}

// IsAfter reports whether a was created after b.
func IsAfter(a, b UUID) bool {
	return CompareUUIDv7(a, b) == 1
}

// Nil is the zero UUID value.
var Nil = uuid.Nil
