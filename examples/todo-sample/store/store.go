package store

import "sync"

// Todo representa una tarea individual.
type Todo struct {
	ID   int
	Text string
	Done bool
}

// TodoStore es el store thread-safe de tareas.
// Se guarda en app.State como el "signal" de la lista de TODOs.
type TodoStore struct {
	mu     sync.RWMutex
	todos  []Todo
	nextID int
}

func New() *TodoStore {
	return &TodoStore{nextID: 1}
}

// Add agrega una nueva tarea.
func (s *TodoStore) Add(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos = append(s.todos, Todo{ID: s.nextID, Text: text})
	s.nextID++
}

// Toggle alterna el estado completado de una tarea.
func (s *TodoStore) Toggle(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.todos {
		if s.todos[i].ID == id {
			s.todos[i].Done = !s.todos[i].Done
			return
		}
	}
}

// Delete elimina una tarea por ID.
func (s *TodoStore) Delete(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, t := range s.todos {
		if t.ID == id {
			s.todos = append(s.todos[:i], s.todos[i+1:]...)
			return
		}
	}
}

// All retorna una copia de todas las tareas (safe para templates).
func (s *TodoStore) All() []Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Todo, len(s.todos))
	copy(result, s.todos)
	return result
}

// Stats retorna conteos útiles para el template.
func (s *TodoStore) Stats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	done := 0
	for _, t := range s.todos {
		if t.Done {
			done++
		}
	}
	return map[string]int{
		"Total":   len(s.todos),
		"Done":    done,
		"Pending": len(s.todos) - done,
	}
}
