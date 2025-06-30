package objectstore

import (
	"crypto/sha512"
	"encoding/hex"
)

// HexEncodedSHA512 takes a byte slice as input and returns its SHA-512 hash as a hex-encoded string.
func HexEncodedSHA512(data []byte) string {
	hash := sha512.Sum512(data)        // Generate SHA-512 hash
	return hex.EncodeToString(hash[:]) // Convert to hex string
}
