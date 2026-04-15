# Security in Thunder Framework

Thunder is designed with security best practices in mind, leveraging Go's robust standard library to protect against common vulnerabilities.

## 1. Cross-Site Scripting (XSS)
Thunder protects against XSS primarily through its rendering engine:
- **Automatic Contextual Escaping**: By using Go's `html/template` package, Thunder automatically escapes all data provided to templates. It is context-aware, meaning it knows whether a variable is being placed in an HTML attribute, a JavaScript block, or a plain text node, and it applies the appropriate escaping for that context.
- **Secure Cookies**: Session cookies are marked as `HttpOnly`, ensuring that even if an attacker manages to execute JavaScript on the page, they cannot access the user's session ID.

## 2. SQL Injection
While Thunder is currently in-memory, it encourages the following practices for database integrations:
- **Parameterized Queries**: Users should always use parameterized queries with Go's `database/sql` package. Go handles the separation of SQL logic and user data at the driver level, making SQL injection impossible when used correctly.
- **No Manual String Concatenation**: The framework's design promotes passing structured data to handlers, discouraging the manual building of SQL strings.

## 3. Cross-Site Request Forgery (CSRF)
Thunder provides multi-layered CSRF protection using the **Double-Submit Cookie** pattern, active by default on all state-changing requests (POST, PUT, DELETE, PATCH).

### How it works
1. **Token Generation**: On the first request, the CSRF middleware generates a 32-byte cryptographically random token and stores it in a `thunder_csrf` cookie (`SameSite=Strict`, `Secure=auto`).
2. **Automatic Injection**: The template preprocessor automatically injects a hidden `_csrf` field before every `</form>` tag. For HTMX requests, a `htmx:configRequest` event listener reads the cookie and sends it as an `X-CSRF-Token` header.
3. **Validation**: On every state-changing request, the middleware compares the cookie token against the submitted token (header or form field) using constant-time comparison (`crypto/subtle`) to prevent timing attacks.
4. **Rejection**: Requests with missing or mismatched tokens receive HTTP 403 Forbidden.

### Additional layers
- **SameSite=Lax** on session cookies prevents cross-site cookie transmission on POST.
- **SameSite=Strict** on the CSRF cookie provides even stricter same-site enforcement.
- **Action Pattern**: The `Action` system encourages POST for state changes, combined with CSRF validation.

### Configuration
```go
app.Run(thunder.AppArgs{
    Port:        8086,
    DisableCSRF: false,                          // default: enabled
    CSRFExempt:  []string{"/webhooks/stripe"},   // paths that skip validation
})
```

### Manual token access
For custom handlers that need the CSRF token (e.g., JSON API responses):
```go
import "thunder/csrf"

token := csrf.Token(r) // reads from request context
```

## 4. Session Security
- **Secure ID Generation**: Session IDs are 16-byte cryptographically secure random values (32-character hex strings), making them virtually impossible to guess or brute-force.
- **Session Expiration**: Automatic background cleanup ensures that stale sessions are pruned, reducing the window of opportunity for session hijacking.
- **HTTPS Enforcement**: The `Secure` flag is automatically applied to cookies if the application is accessed over a TLS/SSL connection.
