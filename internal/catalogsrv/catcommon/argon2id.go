package catcommon

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	// Format version
	formatVersion = 0x01

	// Cryptographic parameters
	saltSize    = 16
	keySize     = 32
	nonceSize   = 12
	memory      = 64 * 1024 // 64 MB
	iterations  = 3
	parallelism = 4

	// Minimum blob size: version(1) + salt(16) + nonce(12) + min ciphertext(1)
	minBlobSize = 1 + saltSize + nonceSize + 1
)

// zero overwrites the given byte slice with zeros
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Derives a 32-byte key from a password and salt using Argon2id
func deriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, iterations, memory, uint8(parallelism), keySize)
}

// validateFormat checks if the blob has a valid format
func validateFormat(blob []byte) error {
	if len(blob) < minBlobSize {
		return fmt.Errorf("invalid blob length: %d (minimum: %d)", len(blob), minBlobSize)
	}

	if blob[0] != formatVersion {
		return fmt.Errorf("unsupported format version: %d", blob[0])
	}

	// Ensure we have at least some ciphertext
	ciphertextLen := len(blob) - (1 + saltSize + nonceSize)
	if ciphertextLen <= 0 {
		return fmt.Errorf("invalid ciphertext length: %d", ciphertextLen)
	}

	return nil
}

// Encrypts raw binary data with a password using Argon2id + AES-GCM
func Encrypt(data []byte, password string) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key := deriveKey([]byte(password), salt)
	defer zero(key) // Zero the key after use

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, nonceSize) // #nosec G407
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Note: AAD support could be added in a future version
	ciphertext := aesgcm.Seal(nil, nonce, data, nil)

	// Format: [version(1B)][salt(16B)][nonce(12B)][ciphertext(N)]
	result := []byte{formatVersion}
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// Decrypts the encrypted blob using the password
func Decrypt(blob []byte, password string) ([]byte, error) {
	if err := validateFormat(blob); err != nil {
		return nil, err
	}

	salt := blob[1 : 1+saltSize]
	nonce := blob[1+saltSize : 1+saltSize+nonceSize]
	ciphertext := blob[1+saltSize+nonceSize:]

	key := deriveKey([]byte(password), salt)
	defer zero(key) // Zero the key after use

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Note: AAD support could be added in a future version
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}
