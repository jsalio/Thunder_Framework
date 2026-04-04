// Package csrf provides Cross-Site Request Forgery protection middleware
// for the Thunder framework using the Double-Submit Cookie pattern.
//
// The middleware generates a random token stored in a cookie (thunder_csrf).
// On state-changing requests (POST, PUT, DELETE, PATCH), it validates that
// the token sent via header (X-CSRF-Token) or form field (_csrf) matches
// the cookie value using constant-time comparison.
package csrf

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"thunder/router"
)

const (
	// CookieName is the name of the CSRF cookie.
	CookieName = "thunder_csrf"

	// HeaderName is the HTTP header used to submit the CSRF token.
	HeaderName = "X-CSRF-Token"

	// FieldName is the form field name used to submit the CSRF token.
	FieldName = "_csrf"

	// tokenBytes is the number of random bytes for token generation (32 bytes = 64 hex chars).
	tokenBytes = 32
)

// contextKey is used to store the CSRF token in the request context.
type contextKey struct{}

// TokenKey is the context key for retrieving the CSRF token from a request.
var TokenKey = contextKey{}

// Config holds CSRF middleware settings.
type Config struct {
	// Exempt is a set of path patterns to exclude from CSRF validation.
	// Matched against r.URL.Path using exact match.
	// Example: "/webhooks/stripe", "/api/public"
	Exempt map[string]bool
}

// GenerateToken creates a cryptographically random CSRF token.
func GenerateToken() string {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		panic("csrf: failed to generate random token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// ValidateToken compares two tokens using constant-time comparison
// to prevent timing attacks. Returns true if they match.
func ValidateToken(cookieToken, submittedToken string) bool {
	if cookieToken == "" || submittedToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(submittedToken)) == 1
}

// safeMethods are HTTP methods that don't require CSRF validation.
var safeMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

// Protect returns a CSRF protection middleware.
// It ensures a CSRF cookie is set on every response and validates
// the token on state-changing requests.
func Protect(cfg ...Config) router.Middleware {
	var exempt map[string]bool
	if len(cfg) > 0 && cfg[0].Exempt != nil {
		exempt = cfg[0].Exempt
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ensure CSRF cookie exists; generate one if missing.
			token := tokenFromCookie(r)
			if token == "" {
				token = GenerateToken()
				http.SetCookie(w, &http.Cookie{
					Name:     CookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // JS/HTMX must read this cookie
					Secure:   r.TLS != nil,
					SameSite: http.SameSiteStrictMode,
					MaxAge:   3600,
				})
			}

			// Store token in request context for template access.
			ctx := r.Context()
			r = r.WithContext(withToken(ctx, token))

			// Safe methods and exempt paths skip validation.
			if safeMethods[r.Method] {
				next.ServeHTTP(w, r)
				return
			}

			if exempt != nil && exempt[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Validate: submitted token must match cookie token.
			submitted := tokenFromRequest(r)
			if !ValidateToken(token, submitted) {
				http.Error(w, "Forbidden - CSRF token invalid", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Token extracts the CSRF token from a request context.
// Returns empty string if not found.
func Token(r *http.Request) string {
	if t, ok := r.Context().Value(TokenKey).(string); ok {
		return t
	}
	return ""
}

// tokenFromCookie reads the CSRF token from the request cookie.
func tokenFromCookie(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// tokenFromRequest extracts the submitted CSRF token from the request,
// checking the header first, then the form field.
func tokenFromRequest(r *http.Request) string {
	// 1. Check header (preferred for HTMX/AJAX).
	if h := r.Header.Get(HeaderName); h != "" {
		return h
	}

	// 2. Check form field (standard HTML forms).
	if err := r.ParseForm(); err == nil {
		if f := r.FormValue(FieldName); f != "" {
			return f
		}
	}

	return ""
}

// withToken stores the CSRF token in the request context.
func withToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, TokenKey, token)
}
