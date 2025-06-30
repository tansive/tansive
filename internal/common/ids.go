// Package common provides utilities for generating unique identifiers and handling common operations.
// It supports various ID types and secure random number generation.
package common

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
)

// IdType represents the type of identifier to generate
type IdType int

// Supported ID types
const (
	ID_TYPE_GENERIC IdType = iota // Generic identifier
	ID_TYPE_TENANT                // Tenant identifier
	ID_TYPE_USER                  // User identifier
	ID_TYPE_PROJECT               // Project identifier
)

// Constants for ID generation
const (
	AIRLINE_CODE_LEN = 6 // Length of the airline-style code portion of IDs

	// Character sets for ID generation
	LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ" // Uppercase letters
	DIGITS  = "0123456789"                 // Numeric digits
	CHARS   = LETTERS + DIGITS             // Combined character set
)

// secureRandomInt generates a cryptographically secure random number between 0 and max.
// Returns an error if random number generation fails.
func secureRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive, got %d", max)
	}
	if max > math.MaxInt32 {
		return 0, fmt.Errorf("max too large: %d", max)
	}

	// Find the largest multiple of max within uint64 to avoid modulo bias
	limit := (math.MaxUint64 / uint64(max)) * uint64(max)

	for {
		var buf [8]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return 0, fmt.Errorf("failed to generate random bytes: %w", err)
		}
		n := binary.BigEndian.Uint64(buf[:])
		if n < limit {
			// Ensure the value fits within int range
			if n > uint64(math.MaxInt) {
				continue
			}
			return int(n % uint64(max)), nil
		}
	}
}

// GetUniqueId generates a unique ID with a prefix based on the type.
// The ID format is a single character prefix followed by an airline-style code.
// Note: This is randomly generated and may not be globally unique.
// Collision probability is approximately 1.5% in 10 million keys.
// Returns an error if ID generation fails.
func GetUniqueId(t IdType) (string, error) {
	code, err := airlineCode(AIRLINE_CODE_LEN)
	if err != nil {
		return "", fmt.Errorf("failed to generate unique ID: %w", err)
	}

	prefix := ""
	switch t {
	case ID_TYPE_TENANT:
		prefix = "T"
	case ID_TYPE_USER:
		prefix = "U"
	case ID_TYPE_PROJECT:
		prefix = "P"
	}

	return prefix + code, nil
}

// airlineCode generates a random alphanumeric string of a given length.
// The first character is always a letter, followed by alphanumeric characters.
// Returns an error if length is invalid or random generation fails.
func airlineCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive, got %d", length)
	}

	result := make([]byte, length)

	letterIdx, err := secureRandomInt(len(LETTERS))
	if err != nil {
		return "", fmt.Errorf("failed to generate first character: %w", err)
	}
	result[0] = LETTERS[letterIdx]

	for i := 1; i < length; i++ {
		idx, err := secureRandomInt(len(CHARS))
		if err != nil {
			return "", fmt.Errorf("failed to generate character at position %d: %w", i, err)
		}
		result[i] = CHARS[idx]
	}

	return string(result), nil
}
