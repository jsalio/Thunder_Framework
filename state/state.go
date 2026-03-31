// Package state provides the global state store for the framework.
// It works similarly to Angular's signal store: a map of values accessible from any component.
package state

import (
	"sync"
	"time"
)

// State is the global state store of the framework.
type State struct {
	mu           sync.RWMutex
	data         map[string]any
	LastAccessed time.Time // New field for session expiration
}

// New creates and initializes a new State store.
func New() *State {
	return &State{
		data:         make(map[string]any),
		LastAccessed: time.Now(),
	}
}

// Set stores a value in the state.
func (s *State) Set(key string, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

// Get retrieves a value from the state. Returns nil if it doesn't exist.
func (s *State) Get(key string) any {
	s.mu.Lock() // Update LastAccessed requires a write lock
	defer s.mu.Unlock()
	s.LastAccessed = time.Now()
	return s.data[key]
}

// Snapshot returns a complete copy of the state as a map.
// It's used to pass the entire state to templates.
func (s *State) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy := make(map[string]any, len(s.data))
	for k, v := range s.data {
		copy[k] = v
	}
	return copy
}

// SessionStore manages multiple State objects, one per session ID.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*State
}

// NewSessionStore creates a new session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*State),
	}
}

// Get retrieves the state for a given session ID. Creates it if it doesn't exist.
func (s *SessionStore) Get(sessionID string) *State {
	s.mu.RLock()
	st, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if ok {
		// Update LastAccessed
		st.mu.Lock()
		st.LastAccessed = time.Now()
		st.mu.Unlock()
		return st
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Double check after lock
	if st, ok = s.sessions[sessionID]; ok {
		return st
	}

	// Limit maximum sessions to prevent memory exhaustion
	if len(s.sessions) >= 5000 {
		// If full, skip creating new ones or you could implement an LRU eviction here.
		// For now, we just return a temporary un-persisted state to allow the request to proceed.
		return New()
	}

	st = New()
	s.sessions[sessionID] = st
	return st
}

// Cleanup removes sessions that haven't been accessed within the given TTL.
func (s *SessionStore) Cleanup(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, st := range s.sessions {
		st.mu.RLock()
		last := st.LastAccessed
		st.mu.RUnlock()

		if now.Sub(last) > ttl {
			delete(s.sessions, id)
		}
	}
}

// Len returns the current number of active sessions.
func (s *SessionStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}
