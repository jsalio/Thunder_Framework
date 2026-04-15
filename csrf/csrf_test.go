package csrf

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	token := GenerateToken()
	if len(token) != tokenBytes*2 {
		t.Fatalf("expected token length %d, got %d", tokenBytes*2, len(token))
	}

	// Tokens must be unique.
	other := GenerateToken()
	if token == other {
		t.Fatal("two generated tokens should not be equal")
	}
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name     string
		cookie   string
		submit   string
		expected bool
	}{
		{"matching tokens", "abc123", "abc123", true},
		{"mismatched tokens", "abc123", "xyz789", false},
		{"empty cookie", "", "abc123", false},
		{"empty submitted", "abc123", "", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateToken(tt.cookie, tt.submit); got != tt.expected {
				t.Errorf("ValidateToken(%q, %q) = %v, want %v", tt.cookie, tt.submit, got, tt.expected)
			}
		})
	}
}

func TestProtect_SetsCookieOnGET(t *testing.T) {
	handler := Protect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == CookieName {
			found = true
			if len(c.Value) != tokenBytes*2 {
				t.Errorf("cookie token length = %d, want %d", len(c.Value), tokenBytes*2)
			}
			if c.HttpOnly {
				t.Error("CSRF cookie should not be HttpOnly (JS needs access)")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Error("CSRF cookie should be SameSite=Strict")
			}
		}
	}
	if !found {
		t.Fatal("CSRF cookie not set on GET request")
	}
}

func TestProtect_GETPassesThrough(t *testing.T) {
	called := false
	handler := Protect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("GET request should pass through without validation")
	}
}

func TestProtect_POSTWithoutToken_Returns403(t *testing.T) {
	handler := Protect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "sometoken"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestProtect_POSTWithValidHeader_Passes(t *testing.T) {
	token := GenerateToken()
	called := false

	handler := Protect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	req.Header.Set(HeaderName, token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("handler should be called with valid CSRF token in header")
	}
}

func TestProtect_POSTWithValidFormField_Passes(t *testing.T) {
	token := GenerateToken()
	called := false

	handler := Protect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	form := url.Values{}
	form.Set(FieldName, token)
	req := httptest.NewRequest(http.MethodPost, "/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("handler should be called with valid CSRF token in form field")
	}
}

func TestProtect_POSTWithWrongToken_Returns403(t *testing.T) {
	handler := Protect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "real-token"})
	req.Header.Set(HeaderName, "wrong-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestProtect_ExemptPath_SkipsValidation(t *testing.T) {
	called := false
	cfg := Config{Exempt: map[string]bool{"/webhooks/stripe": true}}

	handler := Protect(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("exempt path should skip CSRF validation")
	}
}

func TestProtect_PUTAndDELETE_RequireToken(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			handler := Protect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("%s should not pass without token", method)
			}))

			req := httptest.NewRequest(method, "/resource", nil)
			req.AddCookie(&http.Cookie{Name: CookieName, Value: "token"})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s: expected 403, got %d", method, rec.Code)
			}
		})
	}
}

func TestToken_FromContext(t *testing.T) {
	var got string

	handler := Protect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = Token(r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got == "" {
		t.Fatal("Token(r) should return the CSRF token from context")
	}
	if len(got) != tokenBytes*2 {
		t.Errorf("token length = %d, want %d", len(got), tokenBytes*2)
	}
}

func TestToken_WithExistingCookie(t *testing.T) {
	existing := GenerateToken()
	var got string

	handler := Protect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = Token(r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: existing})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != existing {
		t.Errorf("expected existing token %q, got %q", existing, got)
	}
}
