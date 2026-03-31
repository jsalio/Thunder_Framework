# Session Management in Thunder

Thunder provides per-user state isolation using a session-based system.

## 1. How it Works
- **Session ID**: A unique, cryptographically secure 16-byte ID is generated for each new user.
- **Cookies**: This ID is stored in a `thunder_session` cookie.
- **State Store**: The `SessionStore` in `state/state.go` manages isolated `State` objects indexed by this session ID.

## 2. Accessing Session State
Within any component handler or action, you can access the current user's state through `ctx.SessionState`:

```go
func(ctx *component.Ctx) any {
    count := ctx.SessionState.Get("count")
    // ...
}
```

## 3. Security Hardening
Thunder applies several security measures to session cookies:
- **HttpOnly**: Prevents client-side scripts (JS) from accessing the session ID, mitigating XSS attacks.
- **SameSite=Lax**: Restricts the cookie to first-party contexts, protecting against CSRF attacks.
- **Secure**: If the request is over HTTPS, the cookie is marked as secure.
- **MaxAge**: Sessions expire after 1 hour of inactivity.

## 4. Performance and Memory
To avoid memory leaks, a background scavenger process runs every minute to prune sessions that haven't been accessed for more than 1 hour. Additionally, a global limit of 5,000 sessions is enforced.
