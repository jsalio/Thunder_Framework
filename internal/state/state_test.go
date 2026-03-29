package state

import (
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.data == nil {
		t.Fatal("New().data is nil")
	}
}

func TestSetGet(t *testing.T) {
	s := New()
	s.Set("key1", "val1")
	s.Set("key2", 123)

	if v := s.Get("key1"); v != "val1" {
		t.Errorf("expected val1, got %v", v)
	}
	if v := s.Get("key2"); v != 123 {
		t.Errorf("expected 123, got %v", v)
	}
	if v := s.Get("nonexistent"); v != nil {
		t.Errorf("expected nil for nonexistent key, got %v", v)
	}
}

func TestSnapshot(t *testing.T) {
	s := New()
	s.Set("a", 1)
	s.Set("b", 2)

	snap := s.Snapshot()
	if len(snap) != 2 {
		t.Errorf("expected snapshot length 2, got %d", len(snap))
	}
	if snap["a"] != 1 || snap["b"] != 2 {
		t.Errorf("snapshot data mismatch")
	}

	// Verify it's a copy
	s.Set("c", 3)
	if _, ok := snap["c"]; ok {
		t.Error("snapshot should be a copy, but it reflected changes to the original state")
	}
}

func TestConcurrency(t *testing.T) {
	s := New()
	var wg sync.WaitGroup
	workers := 100
	iterations := 1000

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				s.Set("key", j)
				_ = s.Get("key")
				_ = s.Snapshot()
			}
		}(i)
	}
	wg.Wait()
}
