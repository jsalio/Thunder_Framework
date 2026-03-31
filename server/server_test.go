package server

import (
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	// Use a dynamic port (0)
	addr := "127.0.0.1:0"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	errChan := make(chan error, 1)
	go func() {
		err := Start(addr, handler)
		errChan <- err
	}()

	// Wait a moment for the server to start
	time.Sleep(200 * time.Millisecond)

	// Simulate interrupt signal to shut down the server
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	
	// We send SIGTERM to our own process so signal.Notify captures it
	p.Signal(syscall.SIGTERM)

	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Start failed with unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for server to shut down")
	}
}
