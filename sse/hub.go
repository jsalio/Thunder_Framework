// Package sse provides a Server-Sent Events hub for broadcasting
// named events to connected clients, scoped by session ID.
//
// Components emit events via ctx.Emit("event-name"), and the hub
// fans the event out to all SSE connections belonging to that session.
// The client-side JS then re-fetches and swaps the affected components.
package sse

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Event represents a named SSE event broadcast to clients.
type Event struct {
	Name      string // event name, e.g. "cart-updated"
	SessionID string // target session
}

// client is a single SSE connection tied to a session.
type client struct {
	ch        chan Event
	sessionID string
}

// Hub manages SSE client connections and broadcasts events per session.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

// NewHub creates and returns a new SSE Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*client]struct{}),
	}
}

// Subscribe registers a new client for the given session and returns
// its event channel. The caller must call Unsubscribe when done.
func (h *Hub) Subscribe(sessionID string) *client {
	c := &client{
		ch:        make(chan Event, 16), // buffered to avoid blocking broadcasts
		sessionID: sessionID,
	}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

// Unsubscribe removes a client and closes its channel.
func (h *Hub) Unsubscribe(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.ch)
}

// Broadcast sends an event to all clients in the given session.
func (h *Hub) Broadcast(sessionID, eventName string) {
	ev := Event{Name: eventName, SessionID: sessionID}
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c.sessionID == sessionID {
			select {
			case c.ch <- ev:
			default:
				// Drop event if client buffer is full (slow consumer).
			}
		}
	}
}

// ClientCount returns the number of currently connected clients.
// Useful for monitoring and testing.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ClientCountForSession returns the number of clients for a specific session.
func (h *Hub) ClientCountForSession(sessionID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for c := range h.clients {
		if c.sessionID == sessionID {
			count++
		}
	}
	return count
}

// ServeHTTP implements the SSE endpoint handler.
// It streams events to the client using the standard SSE protocol.
// The connection stays open until the client disconnects.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request, sessionID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	c := h.Subscribe(sessionID)
	defer h.Unsubscribe(c)

	// Send initial connection event.
	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	// Keep-alive ticker prevents proxy timeouts.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-c.ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Name, ev.Name)
			flusher.Flush()
		case <-ticker.C:
			// Send a comment as keep-alive.
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
