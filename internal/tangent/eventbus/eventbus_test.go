package eventbus

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	bus := New()
	if bus == nil || bus.subscribers == nil {
		t.Error("New() returned nil or subscribers map is nil")
	}
}

func TestSubscribeAndPublish(t *testing.T) {
	bus := New()
	topic := "test-topic"
	bufferSize := 1

	// Subscribe to topic
	ch, unsubscribe := bus.Subscribe(topic, bufferSize)
	defer unsubscribe()

	// Publish an event
	testData := "test-data"
	bus.Publish(topic, testData, 100*time.Millisecond)

	// Verify the event was received
	select {
	case event := <-ch:
		if event.Topic != topic {
			t.Errorf("expected topic %s, got %s", topic, event.Topic)
		}
		if event.Data != testData {
			t.Errorf("expected data %v, got %v", testData, event.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := New()
	topic := "test-topic"
	bufferSize := 1

	// Create multiple subscribers
	ch1, unsub1 := bus.Subscribe(topic, bufferSize)
	defer unsub1()
	ch2, unsub2 := bus.Subscribe(topic, bufferSize)
	defer unsub2()

	// Publish an event
	testData := "test-data"
	bus.Publish(topic, testData, 100*time.Millisecond)

	// Verify both subscribers received the event
	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case event := <-ch:
			if event.Topic != topic {
				t.Errorf("subscriber %d: expected topic %s, got %s", i, topic, event.Topic)
			}
			if event.Data != testData {
				t.Errorf("subscriber %d: expected data %v, got %v", i, testData, event.Data)
			}
		case <-time.After(200 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := New()
	topic := "test-topic"
	bufferSize := 1

	// Subscribe and then unsubscribe
	ch, unsubscribe := bus.Subscribe(topic, bufferSize)

	// Allow subscription setup
	time.Sleep(10 * time.Millisecond)

	unsubscribe()

	// Allow unsubscribe cleanup
	time.Sleep(10 * time.Millisecond)

	// Try to publish after unsubscribe
	bus.Publish(topic, "test-data", 100*time.Millisecond)

	// Since channel is closed, any receive will return immediately — check if it's closed
	_, ok := <-ch
	if ok {
		t.Errorf("channel is still open after unsubscribe")
	}
}

func TestShutdown(t *testing.T) {
	bus := New()
	topic := "test-topic"
	bufferSize := 1

	// Subscribe to multiple topics
	ch1, unsub1 := bus.Subscribe(topic, bufferSize)
	defer unsub1()
	ch2, unsub2 := bus.Subscribe("another-topic", bufferSize)
	defer unsub2()

	// Shutdown the bus
	bus.Shutdown()

	// Try to publish after shutdown
	bus.Publish(topic, "test-data", 100*time.Millisecond)

	// Ensure both channels are closed
	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case _, ok := <-ch:
			if ok {
				t.Errorf("subscriber %d: channel should be closed after shutdown", i)
			}
		case <-time.After(200 * time.Millisecond):
			t.Errorf("subscriber %d: did not close channel after shutdown", i)
		}
	}
}

func TestPublishTimeout(t *testing.T) {
	bus := New()
	topic := "test-topic"
	bufferSize := 1

	// Subscribe with a small buffer
	ch, unsubscribe := bus.Subscribe(topic, bufferSize)
	defer unsubscribe()

	// Fill the buffer
	bus.Publish(topic, "first", 100*time.Millisecond)

	// Try to publish with a short timeout
	bus.Publish(topic, "second", 1*time.Millisecond)

	// Verify only the first event was received
	select {
	case event := <-ch:
		if event.Data != "first" {
			t.Errorf("expected first event, got %v", event.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for first event")
	}

	// Verify no second event
	select {
	case event := <-ch:
		t.Errorf("unexpected second event received: %v", event)
	case <-time.After(200 * time.Millisecond):
		// This is the expected case
	}
}

func TestContextCancellation(t *testing.T) {
	bus := New()
	topic := "test-topic"
	bufferSize := 1

	// Create a subscriber with a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan Event, bufferSize)
	sub := &Subscriber{
		ID:         "test-sub",
		Topic:      topic,
		BufferSize: bufferSize,
		Channel:    ch,
		Context:    ctx,
		Cancel:     cancel,
	}

	// Add subscriber to the bus
	bus.Lock()
	if _, ok := bus.subscribers[topic]; !ok {
		bus.subscribers[topic] = make(map[string]*Subscriber)
	}
	bus.subscribers[topic][sub.ID] = sub
	bus.Unlock()

	// Cancel the context
	cancel()

	// Try to publish
	bus.Publish(topic, "test-data", 100*time.Millisecond)

	// Verify no event is received
	select {
	case event := <-ch:
		t.Errorf("received event after context cancellation: %v", event)
	case <-time.After(200 * time.Millisecond):
		// This is the expected case
	}
}

func TestPublishNonBlocking(t *testing.T) {
	bus := New()
	topic := "test-topic"
	bufferSize := 1

	// Create a slow subscriber that never reads from the channel
	ch, unsubscribe := bus.Subscribe(topic, bufferSize)
	defer unsubscribe()

	// Fill the subscriber's buffer
	bus.Publish(topic, "first", 100*time.Millisecond)

	// Try to publish multiple events in quick succession
	// These should not block even though the subscriber is not reading
	start := time.Now()
	for i := 0; i < 10; i++ {
		bus.Publish(topic, fmt.Sprintf("event-%d", i), 1*time.Millisecond)
	}
	duration := time.Since(start)

	// Verify that publishing was reasonably fast
	// Each publish operation has a 1ms timeout, so 10 operations should take at most ~10ms
	// We add some buffer for system scheduling
	if duration > 50*time.Millisecond {
		t.Errorf("publishing was too slow (took %v), expected faster non-blocking behavior", duration)
	}

	// Verify only the first event was received (buffer was full)
	select {
	case event := <-ch:
		if event.Data != "first" {
			t.Errorf("expected first event, got %v", event.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for first event")
	}

	// Verify no more events were received
	select {
	case event := <-ch:
		t.Errorf("unexpected event received: %v", event)
	case <-time.After(200 * time.Millisecond):
		// This is the expected case
	}
}

func TestCloseTopic(t *testing.T) {
	bus := New()
	topic := "test-topic"
	bufferSize := 1

	// Create multiple subscribers for the same topic
	ch1, unsub1 := bus.Subscribe(topic, bufferSize)
	defer unsub1()
	ch2, unsub2 := bus.Subscribe(topic, bufferSize)
	defer unsub2()

	// Create a subscriber for a different topic
	otherCh, otherUnsub := bus.Subscribe("other-topic", bufferSize)
	defer otherUnsub()

	// Publish an event to both topics
	bus.Publish(topic, "test-data", 100*time.Millisecond)
	bus.Publish("other-topic", "other-data", 100*time.Millisecond)

	// Drain initial messages
	<-ch1
	<-ch2

	// Close the topic
	bus.CloseTopic(topic)

	// Allow cleanup to complete
	time.Sleep(10 * time.Millisecond)

	// Verify both channels for the closed topic are closed
	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case _, ok := <-ch:
			if ok {
				t.Errorf("subscriber %d: channel should be closed after CloseTopic", i)
			}
		case <-time.After(200 * time.Millisecond):
			t.Errorf("subscriber %d: channel read timed out after CloseTopic", i)
		}
	}

	// Verify the other topic's subscriber is still active
	select {
	case event, ok := <-otherCh:
		if !ok {
			t.Error("expected otherCh to be open, but it was closed")
		} else if event.Data != "other-data" {
			t.Errorf("expected other-data, got %v", event.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for event on other topic")
	}

	// Try to publish again to the closed topic
	bus.Publish(topic, "new-data", 100*time.Millisecond)

	// Verify no new events are received on closed channels
	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case event, ok := <-ch:
			if ok {
				t.Errorf("subscriber %d: received event after CloseTopic: %+v", i, event)
			}
		case <-time.After(200 * time.Millisecond):
			// Expected path
		}
	}
}

func TestWildcardMatching(t *testing.T) {
	bus := New()
	bufferSize := 1

	// Test cases for wildcard matching
	testCases := []struct {
		pattern     string
		topic       string
		shouldMatch bool
	}{
		{"user.*", "user.login", true},
		{"user.*", "user.logout", true},
		{"user.*", "user.profile.update", false}, // Different number of segments
		{"user.*.update", "user.profile.update", true},
		{"user.*.update", "user.settings.update", true},
		{"user.*.update", "user.update", false}, // Missing middle segment
		{"*.update", "user.update", true},
		{"*.update", "profile.update", true},
		{"*.update", "update", false}, // Missing first segment
		{"user.*.*", "user.profile.update", true},
		{"user.*.*", "user.settings.delete", true},
		{"user.*.*", "user.profile", false}, // Missing last segment
		{"*.*.*", "user.profile.update", true},
		{"*.*.*", "any.topic.here", true},
		{"*.*.*", "too.many.segments.here", false}, // Too many segments
		{"*", "any.topic.here", true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("pattern=%s,topic=%s", tc.pattern, tc.topic), func(t *testing.T) {
			// Subscribe with the pattern
			ch, unsubscribe := bus.Subscribe(tc.pattern, bufferSize)
			defer unsubscribe()

			// Publish to the topic
			testData := "test-data"
			bus.Publish(tc.topic, testData, 100*time.Millisecond)

			// Check if we received the event based on whether it should match
			select {
			case event := <-ch:
				if !tc.shouldMatch {
					t.Errorf("unexpected match: pattern %s matched topic %s", tc.pattern, tc.topic)
				}
				if event.Topic != tc.topic {
					t.Errorf("expected topic %s, got %s", tc.topic, event.Topic)
				}
				if event.Data != testData {
					t.Errorf("expected data %v, got %v", testData, event.Data)
				}
			case <-time.After(200 * time.Millisecond):
				if tc.shouldMatch {
					t.Errorf("expected match but didn't receive event: pattern %s, topic %s", tc.pattern, tc.topic)
				}
				// If we shouldn't match, this is the expected case
			}
		})
	}
}

func TestCloseAllForPattern(t *testing.T) {
	bus := New()
	bufferSize := 1

	// Create subscribers for different topics
	userLoginCh, unsub1 := bus.Subscribe("user.login", bufferSize)
	defer unsub1()
	userLogoutCh, unsub2 := bus.Subscribe("user.logout", bufferSize)
	defer unsub2()
	userProfileCh, unsub3 := bus.Subscribe("user.profile", bufferSize)
	defer unsub3()
	otherCh, unsub4 := bus.Subscribe("other.topic", bufferSize)
	defer unsub4()

	// Publish initial events to all topics
	bus.Publish("user.login", "login-data", 100*time.Millisecond)
	bus.Publish("user.logout", "logout-data", 100*time.Millisecond)
	bus.Publish("user.profile", "profile-data", 100*time.Millisecond)
	bus.Publish("other.topic", "other-data", 100*time.Millisecond)

	// Drain initial messages
	<-userLoginCh
	<-userLogoutCh
	<-userProfileCh
	<-otherCh

	// Close all user.* topics
	bus.CloseAllForPattern("user.*")

	// Allow cleanup to complete
	time.Sleep(10 * time.Millisecond)

	// Verify user.* channels are closed
	for i, ch := range []<-chan Event{userLoginCh, userLogoutCh, userProfileCh} {
		select {
		case _, ok := <-ch:
			if ok {
				t.Errorf("user.* subscriber %d: channel should be closed after CloseAllForPattern", i)
			}
		case <-time.After(200 * time.Millisecond):
			t.Errorf("user.* subscriber %d: channel read timed out after CloseAllForPattern", i)
		}
	}

	// Publish a new event to other.topic to verify it's still active
	bus.Publish("other.topic", "new-other-data", 100*time.Millisecond)

	// Verify other topic's subscriber is still active
	select {
	case event, ok := <-otherCh:
		if !ok {
			t.Error("expected otherCh to be open, but it was closed")
		} else if event.Data != "new-other-data" {
			t.Errorf("expected new-other-data, got %v", event.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for event on other topic")
	}

	// Try to publish again to the closed topics
	bus.Publish("user.login", "new-login-data", 100*time.Millisecond)
	bus.Publish("user.logout", "new-logout-data", 100*time.Millisecond)
	bus.Publish("user.profile", "new-profile-data", 100*time.Millisecond)

	// Verify no new events are received on closed channels
	for i, ch := range []<-chan Event{userLoginCh, userLogoutCh, userProfileCh} {
		select {
		case event, ok := <-ch:
			if ok {
				t.Errorf("user.* subscriber %d: received event after CloseAllForPattern: %+v", i, event)
			}
		case <-time.After(200 * time.Millisecond):
			// Expected path
		}
	}
}
