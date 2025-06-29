package uuid

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUUID7(t *testing.T) {
	id := UUID7()
	assert.NotEqual(t, uuid.Nil, id)
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestNewRandom(t *testing.T) {
	id, err := NewRandom()
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestNew(t *testing.T) {
	id := New()
	assert.NotEqual(t, uuid.Nil, id)
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestParse(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	id, err := Parse(validUUID)
	assert.NoError(t, err)
	assert.Equal(t, validUUID, id.String())

	invalidUUID := "invalid-uuid"
	_, err = Parse(invalidUUID)
	assert.Error(t, err)
}

func TestMustParse(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	id := MustParse(validUUID)
	assert.Equal(t, validUUID, id.String())

	assert.Panics(t, func() {
		MustParse("invalid-uuid")
	})
}

func TestIsUUIDv7(t *testing.T) {
	id := UUID7()
	assert.True(t, IsUUIDv7(id))

	nonV7 := uuid.New()
	assert.False(t, IsUUIDv7(nonV7))
}

func TestGetTimestampFromUUID(t *testing.T) {
	id := UUID7()
	timestamp := GetTimestampFromUUID(id)

	// The timestamp should be within a reasonable range of the current time
	now := time.Now()
	diff := now.Sub(timestamp)
	assert.True(t, diff >= -time.Second && diff <= time.Second)
}

func TestCompareUUIDv7(t *testing.T) {
	id1 := UUID7()
	id2 := UUID7()

	// id1 should be before id2
	assert.Equal(t, -1, CompareUUIDv7(id1, id2))
	assert.Equal(t, 1, CompareUUIDv7(id2, id1))
	assert.Equal(t, 0, CompareUUIDv7(id1, id1))
}

func TestIsBeforeAndAfter(t *testing.T) {
	id1 := UUID7()
	time.Sleep(time.Millisecond)
	id2 := UUID7()

	assert.True(t, IsBefore(id1, id2))
	assert.True(t, IsAfter(id2, id1))
	assert.False(t, IsBefore(id1, id1))
	assert.False(t, IsAfter(id1, id1))
}
