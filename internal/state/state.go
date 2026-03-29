package state

import "sync"

// State es el store de estado global del framework.
// Funciona como el equivalente al signal store de Angular:
// un mapa de valores accesibles desde cualquier componente.
type State struct {
	mu   sync.RWMutex
	data map[string]any
}

func New() *State {
	return &State{
		data: make(map[string]any),
	}
}

// Set establece un valor en el estado.
func (s *State) Set(key string, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

// Get retorna un valor del estado. Retorna nil si no existe.
func (s *State) Get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}

// Snapshot retorna una copia completa del estado como map.
// Se usa para pasar el estado completo a los templates.
func (s *State) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy := make(map[string]any, len(s.data))
	for k, v := range s.data {
		copy[k] = v
	}
	return copy
}
