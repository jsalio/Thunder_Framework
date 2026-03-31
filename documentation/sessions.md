# Session Management in Thunder

Thunder provides per-user state isolation using a session-based system.

## 1. How it Works
- **Session ID**: A unique, cryptographically secure 16-byte ID is generated for each new user.
- **Cookies**: This ID is stored in a `thunder_session` cookie.
- **State Store**: The `SessionStore` in `state/state.go` manages isolated `State` objects indexed by this session ID.

## 2. Who is the Manager?
The responsibility is shared between the **Framework** and the **Developer (User)**:

### Framework Responsibilities (The Infrastructure)
Thunder manages the session lifecycle automatically:
- **Identification**: It reads the session cookie from every incoming request.
- **Creation**: It generates new IDs and sets cookies for first-time visitors.
- **Injection**: It provides the correct `SessionState` to each `component.Ctx` before your handler runs.
- **Cleanup**: A background "scavenger" process automatically deletes sessions that haven't been used for 1 hour, protecting the server's memory.

### Developer/User Responsibilities (The Data)
You decide exactly **what** to store in the session and how to use it:
- **Data Control**: You use `ctx.SessionState.Set("key", value)` to store any Go type (int, string, struct).
- **Business Logic**: Your component `Handler` or `Action` reads that data using `ctx.SessionState.Get("key")` to determine what to render (e.g., shopping cart items, login status).
- **Persistence**: You decide when to clear data using `ctx.SessionState.Set("key", nil)`.

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
