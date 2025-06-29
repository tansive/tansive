package eventlogger

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/tangent/eventbus"
)

func TestLogWriter(t *testing.T) {
	bus := eventbus.New()
	defer bus.Shutdown()

	writer := &LogWriter{
		Bus:   bus,
		Topic: "test-logs",
	}

	ch, unsubscribe := bus.Subscribe("test-logs", 1)
	defer unsubscribe()

	testMsg := []byte("test log message")
	n, err := writer.Write(testMsg)

	assert.NoError(t, err)
	assert.Equal(t, len(testMsg), n)

	select {
	case event := <-ch:
		data, ok := event.Data.([]byte)
		assert.True(t, ok, "event.Data should be a byte slice")
		assert.Equal(t, testMsg, data)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for log message")
	}
}

func TestNewLogger(t *testing.T) {
	bus := eventbus.New()
	defer bus.Shutdown()

	logger := NewLogger(bus, "test-logs")
	assert.NotNil(t, logger)

	ch, unsubscribe := bus.Subscribe("test-logs", 1)
	defer unsubscribe()

	logger.Info().Msg("test info message")

	select {
	case event := <-ch:
		data, ok := event.Data.([]byte)
		assert.True(t, ok, "event.Data should be a byte slice")
		msg := string(data)
		fmt.Println(msg)
		assert.Contains(t, msg, "test info message")
		assert.Contains(t, msg, "\"level\":\"info\"")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for log message")
	}
}
