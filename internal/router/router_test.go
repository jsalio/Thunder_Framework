package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterMethods(t *testing.T) {
	r := New()
	methods := []struct {
		method string
		fn     func(string, http.HandlerFunc)
	}{
		{"GET", r.GET},
		{"POST", r.POST},
		{"PUT", r.PUT},
		{"DELETE", r.DELETE},
		{"PATCH", r.PATCH},
	}

	for _, m := range methods {
		m.fn("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(m.method, "/test", nil)
		rr := httptest.NewRecorder()
		r.Handler().ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected OK for method %s, got %d", m.method, rr.Code)
		}
	}
}

func TestRouterMiddlewareOrder(t *testing.T) {
	r := New()
	var order []string

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1")
			next.ServeHTTP(w, r)
		})
	}
	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2")
			next.ServeHTTP(w, r)
		})
	}

	r.Use(m1)
	r.Use(m2)
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	r.Handler().ServeHTTP(rr, req)

	expected := []string{"m1", "m2", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("expected %s at index %d, got %s", v, i, order[i])
		}
	}
}

func TestRouterHandle(t *testing.T) {
	r := New()
	r.Handle("/custom", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest("GET", "/custom", nil)
	rr := httptest.NewRecorder()
	r.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected %d, got %d", http.StatusCreated, rr.Code)
	}
}
