// Package eventbus provides an in-memory publish/subscribe event bus for inter-goroutine communication.
// It implements a thread-safe event system with topic-based routing and pattern matching capabilities.
// The package supports buffered channels, timeout-based publishing, and graceful shutdown for concurrent applications.
package eventbus

// Implements a trivial in-memory pub/sub event bus. Used for communication between the different goroutines.  There are opportunities for
// some optimization, but there shouldn't be bottlnecks for a hundred or so skills running in parallel.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Event represents a single event in the event bus.
// Contains the topic and associated data for event routing and processing.
type Event struct {
	Topic string // event topic for routing
	Data  any    // event data payload
}

// Subscriber represents an event subscription with buffered channel and lifecycle management.
// Provides thread-safe event delivery with timeout and cancellation support.
type Subscriber struct {
	ID         string             // unique subscriber identifier
	Topic      string             // subscribed topic pattern
	BufferSize int                // channel buffer size
	Channel    chan Event         // event delivery channel
	Context    context.Context    // cancellation context
	Cancel     context.CancelFunc // context cancellation function

	mu     sync.Mutex // protects closed flag
	closed bool       // indicates if subscriber is closed
}

// SafeSend attempts to send an event to the subscriber's channel.
// Returns true if the event was sent successfully, false if subscriber is closed or channel is full.
func (s *Subscriber) SafeSend(event Event) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false
	}

	select {
	case s.Channel <- event:
		return true
	default:
		return false
	}
}

// TimedSend attempts to send an event with a specified timeout.
// Returns true if the event was sent successfully within the timeout period.
func (s *Subscriber) TimedSend(event Event, timeout time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false
	}

	select {
	case s.Channel <- event:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Close gracefully shuts down the subscriber.
// Cancels the context and closes the event channel.
func (s *Subscriber) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		s.Cancel()
		close(s.Channel)
	}
}

// EventBus provides the main event bus implementation with topic-based routing.
// Manages subscribers, handles event publishing, and supports pattern matching for topic routing.
type EventBus struct {
	sync.RWMutex
	subscribers map[string]map[string]*Subscriber // topic -> subscriberID -> Subscriber
	counter     uint64                            // atomic counter for subscriber ID generation
}

// New creates a new EventBus instance.
// Returns an initialized event bus ready for subscription and publishing.
func New() *EventBus {
	return &EventBus{
		subscribers: make(map[string]map[string]*Subscriber),
	}
}

// Subscribe creates a new subscriber for the given topic and returns the event channel and unsubscribe function.
// The bufferSize parameter controls the channel buffer capacity for event delivery.
// Returns a receive-only channel for events and a function to unsubscribe.
func (bus *EventBus) Subscribe(topic string, bufferSize int) (<-chan Event, func()) {
	id := fmt.Sprintf("sub-%d", atomic.AddUint64(&bus.counter, 1))

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan Event, bufferSize)

	sub := &Subscriber{
		ID:         id,
		Topic:      topic,
		BufferSize: bufferSize,
		Channel:    ch,
		Context:    ctx,
		Cancel:     cancel,
	}

	bus.Lock()
	defer bus.Unlock()

	if _, ok := bus.subscribers[topic]; !ok {
		bus.subscribers[topic] = make(map[string]*Subscriber)
	}
	bus.subscribers[topic][id] = sub

	unsubscribe := func() {
		bus.Lock()
		defer bus.Unlock()

		if subMap, ok := bus.subscribers[topic]; ok {
			if s, ok := subMap[id]; ok {
				s.Close()
				delete(subMap, id)
				if len(subMap) == 0 {
					delete(bus.subscribers, topic)
				}
			}
		}
	}

	return ch, unsubscribe
}

// CloseTopic removes all subscribers for a given topic.
// Gracefully closes all subscribers and removes the topic from the bus.
func (bus *EventBus) CloseTopic(topic string) {
	bus.Lock()
	defer bus.Unlock()

	if subs, ok := bus.subscribers[topic]; ok {
		for _, sub := range subs {
			sub.Close()
		}
		delete(bus.subscribers, topic)
	}
}

// CloseAllForPattern removes all subscribers matching the given pattern.
// Uses pattern matching to identify and close subscribers for multiple topics.
func (bus *EventBus) CloseAllForPattern(pattern string) {
	bus.Lock()
	defer bus.Unlock()

	for topic, subMap := range bus.subscribers {
		if matchTopic(pattern, topic) {
			for _, sub := range subMap {
				sub.Close()
			}
		}
	}
}

// Publish sends an event to all subscribers of a topic.
// Non-blocking; will drop events for slow subscribers based on timeout.
// Uses pattern matching to route events to appropriate subscribers.
func (bus *EventBus) Publish(topic string, data any, timeout time.Duration) {
	event := Event{Topic: topic, Data: data}

	bus.RLock()
	defer bus.RUnlock()

	for pattern, subMap := range bus.subscribers {
		if matchTopic(pattern, topic) {
			for _, sub := range subMap {
				select {
				case <-sub.Context.Done():
					continue
				default:
					sub.TimedSend(event, timeout)
				}
			}
		}
	}
}

// Shutdown gracefully closes all subscribers and clears the bus.
// Ensures all resources are properly cleaned up and channels are closed.
func (bus *EventBus) Shutdown() {
	bus.Lock()
	defer bus.Unlock()

	for _, subs := range bus.subscribers {
		for _, sub := range subs {
			sub.Close()
		}
	}
	bus.subscribers = make(map[string]map[string]*Subscriber)
}

// matchTopic determines if a topic matches a pattern.
// Supports exact matches and wildcard patterns with dot-separated components.
// Returns true if the topic matches the pattern, false otherwise.
func matchTopic(pattern, topic string) bool {
	if pattern == "" || topic == "" {
		return false
	}
	if pattern == "*" || pattern == topic {
		return true
	}
	patternParts := strings.Split(pattern, ".")
	topicParts := strings.Split(topic, ".")

	if len(patternParts) != len(topicParts) {
		return false
	}

	for i := 0; i < len(patternParts); i++ {
		if patternParts[i] == "*" {
			continue
		}
		if patternParts[i] != topicParts[i] {
			return false
		}
	}
	return true
}
