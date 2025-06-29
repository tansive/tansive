package catcommon

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		password string
		wantErr  bool
	}{
		{
			name:     "empty data",
			data:     []byte{},
			password: "test123",
			wantErr:  true,
		},
		{
			name:     "simple text",
			data:     []byte("Hello, World!"),
			password: "test123",
			wantErr:  false,
		},
		{
			name:     "binary data",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
			password: "test123",
			wantErr:  false,
		},
		{
			name:     "long text",
			data:     bytes.Repeat([]byte("This is a long text that needs to be encrypted. "), 100),
			password: "test123",
			wantErr:  false,
		},
		{
			name:     "special characters",
			data:     []byte("!@#$%^&*()_+-=[]{}|;:,.<>?"),
			password: "test123",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption
			encrypted, err := Encrypt(tt.data, tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return // Skip decryption tests for error cases
			}

			// Verify encrypted data is different from input
			if bytes.Equal(encrypted, tt.data) {
				t.Error("Encrypted data is identical to input data")
			}

			// Test decryption with correct password
			decrypted, err := Decrypt(encrypted, tt.password)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify decrypted data matches input
			if !bytes.Equal(decrypted, tt.data) {
				t.Errorf("Decrypted data = %v, want %v", decrypted, tt.data)
			}

			// Test decryption with wrong password
			wrongPassword := "wrong" + tt.password
			_, err = Decrypt(encrypted, wrongPassword)
			if err == nil {
				t.Error("Decrypt() with wrong password should fail")
			}
		})
	}
}

func TestFormatValidation(t *testing.T) {
	tests := []struct {
		name    string
		blob    []byte
		wantErr bool
	}{
		{
			name:    "empty blob",
			blob:    []byte{},
			wantErr: true,
		},
		{
			name:    "too short",
			blob:    []byte{0x01, 0x02, 0x03},
			wantErr: true,
		},
		{
			name:    "wrong version",
			blob:    make([]byte, minBlobSize),
			wantErr: true,
		},
		{
			name:    "valid format",
			blob:    append([]byte{formatVersion}, make([]byte, minBlobSize-1)...),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFormat(tt.blob)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptEmptyData(t *testing.T) {
	_, err := Encrypt([]byte{}, "password")
	if err == nil {
		t.Error("Encrypt() with empty data should fail")
	}
}

func TestDecryptTamperedData(t *testing.T) {
	// Encrypt some data
	data := []byte("test data")
	encrypted, err := Encrypt(data, "password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Tamper with the encrypted data
	encrypted[0] = 0xFF // Change version byte
	_, err = Decrypt(encrypted, "password")
	if err == nil {
		t.Error("Decrypt() with tampered version should fail")
	}

	// Restore version and tamper with salt
	encrypted[0] = formatVersion
	encrypted[1] = 0xFF // Change first byte of salt
	_, err = Decrypt(encrypted, "password")
	if err == nil {
		t.Error("Decrypt() with tampered salt should fail")
	}
}

func TestKeyZeroing(t *testing.T) {
	// This is a best-effort test since we can't guarantee memory zeroing
	// in Go, but we can verify the function exists and runs
	key := []byte{1, 2, 3, 4, 5}
	zero(key)
	for _, b := range key {
		if b != 0 {
			t.Error("zero() did not zero all bytes")
		}
	}
}

// The following benchmarks exist for curiosity and don't matter, since we currently decrypt once and cache the
// signing key. KMS should replace this.
func BenchmarkEncrypt(b *testing.B) {
	data := []byte("This is a test message that will be encrypted multiple times")
	password := "test123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Encrypt(data, password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	data := []byte("This is a test message that will be encrypted and decrypted multiple times")
	password := "test123"

	encrypted, err := Encrypt(data, password)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Decrypt(encrypted, password)
		if err != nil {
			b.Fatal(err)
		}
	}
}
