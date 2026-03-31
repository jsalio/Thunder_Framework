// Package state provides the global state store for the framework.
// It works similarly to Angular's signal store: a map of values accessible from any component.
package state

import "sync"

// State is the global state store of the framework.
type State struct {
	mu   sync.RWMutex
	data map[string]any
}

// New creates and initializes a new State store.
func New() *State {
	return &State{
		data: make(map[string]any),
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
	s.mu.RLock()
	defer s.mu.RUnlock()
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
