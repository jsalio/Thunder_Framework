package sse

import (
	"testing"
	"time"
)

func TestHub_SubscribeUnsubscribe(t *testing.T) {
	h := NewHub()
	sessionID := "test-session"

	c := h.Subscribe(sessionID)
	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", h.ClientCount())
	}
	if h.ClientCountForSession(sessionID) != 1 {
		t.Errorf("expected 1 client for session, got %d", h.ClientCountForSession(sessionID))
	}

	h.Unsubscribe(c)
	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients after unsubscribe, got %d", h.ClientCount())
	}
}

func TestHub_Broadcast(t *testing.T) {
	h := NewHub()
	s1 := "session-1"
	s2 := "session-2"

	c1a := h.Subscribe(s1)
	c1b := h.Subscribe(s1)
	c2 := h.Subscribe(s2)

	defer h.Unsubscribe(c1a)
	defer h.Unsubscribe(c1b)
	defer h.Unsubscribe(c2)

	eventName := "test-event"
	h.Broadcast(s1, eventName)

	// c1a and c1b should receive the event
	select {
	case ev := <-c1a.ch:
		if ev.Name != eventName {
			t.Errorf("c1a: expected event %s, got %s", eventName, ev.Name)
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("c1a timed out waiting for event")
	}

	select {
	case ev := <-c1b.ch:
		if ev.Name != eventName {
			t.Errorf("c1b: expected event %s, got %s", eventName, ev.Name)
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("c1b timed out waiting for event")
	}

	// c2 should NOT receive the event
	select {
	case ev := <-c2.ch:
		t.Errorf("c2 received unexpected event: %s", ev.Name)
	case <-time.After(50 * time.Millisecond):
		// Success: timed out as expected
	}
}

func TestHub_SlowConsumer(t *testing.T) {
	h := NewHub()
	sessionID := "slow-session"
	
	// Create client with small buffer to test non-blocking broadcast
	c := h.Subscribe(sessionID)
	defer h.Unsubscribe(c)

	// Fill buffer (default is 16)
	for i := 0; i < 20; i++ {
		h.Broadcast(sessionID, "event")
	}

	// HUB should not be blocked.
	// We should be able to read some events.
	count := 0
	for {
		select {
		case <-c.ch:
			count++
		default:
			goto done
		}
	}
done:
	if count > 16 {
		t.Errorf("read %d events, but buffer size is 16", count)
	}
}
