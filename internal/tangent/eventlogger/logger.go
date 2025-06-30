// Package eventlogger provides logging functionality that integrates with the event bus system.
// It implements zerolog-compatible writers that publish log messages to event bus topics,
// enabling distributed logging and real-time log streaming across the application.
package eventlogger

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/tansive/tansive/internal/tangent/eventbus"
)

// LogWriter is a zerolog-compatible writer that sends logs to an EventBus topic.
// Implements the io.Writer interface to integrate with zerolog logging system.
type LogWriter struct {
	Bus   *eventbus.EventBus // event bus for log publishing
	Topic string             // topic for log message routing
}

// Write publishes a log message to the specified topic on the EventBus.
// Copies the input bytes to avoid data races and publishes with a timeout.
// Returns the number of bytes written and any error encountered during publishing.
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	dup := make([]byte, len(p))
	copy(dup, p)
	lw.Bus.Publish(lw.Topic, dup, 100*time.Millisecond)
	return len(p), nil
}

// NewLogger creates a zerolog.Logger that publishes to the given EventBus topic.
// Returns a configured logger with timestamp field and event bus integration.
func NewLogger(bus *eventbus.EventBus, topic string) zerolog.Logger {
	return zerolog.New(&LogWriter{
		Bus:   bus,
		Topic: topic,
	}).With().Timestamp().Logger()
}
