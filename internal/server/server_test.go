package server

import (
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	// Usar un puerto dinámico (0)
	addr := "127.0.0.1:0"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	errChan := make(chan error, 1)
	go func() {
		err := Start(addr, handler)
		errChan <- err
	}()

	// Esperar un momento a que el servidor arranque
	time.Sleep(200 * time.Millisecond)

	// Simular señal de interrupción para apagar el servidor
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	
	// Enviamos SIGTERM a nuestro propio proceso para que signal.Notify lo capture
	p.Signal(syscall.SIGTERM)

	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Start falló con error inesperado: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout esperando que el servidor se apagara")
	}
}
