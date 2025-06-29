package common

import (
	"math"
	"strings"
	"testing"
)

func TestGetUniqueId(t *testing.T) {
	tests := []struct {
		name    string
		idType  IdType
		wantLen int
		prefix  string
	}{
		{
			name:    "Tenant ID",
			idType:  ID_TYPE_TENANT,
			wantLen: AIRLINE_CODE_LEN + 1,
			prefix:  "T",
		},
		{
			name:    "User ID",
			idType:  ID_TYPE_USER,
			wantLen: AIRLINE_CODE_LEN + 1,
			prefix:  "U",
		},
		{
			name:    "Project ID",
			idType:  ID_TYPE_PROJECT,
			wantLen: AIRLINE_CODE_LEN + 1,
			prefix:  "P",
		},
		{
			name:    "Generic ID",
			idType:  ID_TYPE_GENERIC,
			wantLen: AIRLINE_CODE_LEN,
			prefix:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetUniqueId(tt.idType)
			if err != nil {
				t.Errorf("GetUniqueId() error = %v", err)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("GetUniqueId() length = %v, want %v", len(got), tt.wantLen)
			}
			if tt.prefix != "" && !strings.HasPrefix(got, tt.prefix) {
				t.Errorf("GetUniqueId() prefix = %v, want %v", got[:1], tt.prefix)
			}
		})
	}
}

func TestAirlineCode(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "Valid length",
			length:  AIRLINE_CODE_LEN,
			wantErr: false,
		},
		{
			name:    "Zero length",
			length:  0,
			wantErr: true,
		},
		{
			name:    "Negative length",
			length:  -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := airlineCode(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("airlineCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != tt.length {
					t.Errorf("airlineCode() length = %v, want %v", len(got), tt.length)
				}
				// First character should be a letter
				if !strings.Contains(LETTERS, string(got[0])) {
					t.Errorf("airlineCode() first character = %v, want a letter", got[0])
				}
				// All characters should be alphanumeric
				for _, c := range got {
					if !strings.Contains(CHARS, string(c)) {
						t.Errorf("airlineCode() contains invalid character: %v", c)
					}
				}
			}
		})
	}
}

func TestSecureRandomInt(t *testing.T) {
	tests := []struct {
		name    string
		max     int
		wantErr bool
	}{
		{
			name:    "Valid max",
			max:     100,
			wantErr: false,
		},
		{
			name:    "Zero max",
			max:     0,
			wantErr: true,
		},
		{
			name:    "Negative max",
			max:     -1,
			wantErr: true,
		},
		{
			name:    "Too large max",
			max:     math.MaxInt32 + 1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := secureRandomInt(tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("secureRandomInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got < 0 || got >= tt.max {
					t.Errorf("secureRandomInt() = %v, want in range [0, %v)", got, tt.max)
				}
			}
		})
	}
}

func TestGetUniqueIdUniqueness(t *testing.T) {
	// Generate multiple IDs and check for uniqueness
	generated := make(map[string]bool)
	numIDs := 1000

	for i := 0; i < numIDs; i++ {
		id, err := GetUniqueId(ID_TYPE_USER)
		if err != nil {
			t.Errorf("GetUniqueId() error = %v", err)
			return
		}
		if generated[id] {
			t.Errorf("GetUniqueId() generated duplicate ID: %v", id)
			return
		}
		generated[id] = true
	}
}
