# Concurrency and Connection Management in Thunder

The Thunder Framework is designed to handle multiple simultaneous users efficiently by leveraging Go's native concurrency model and thread-safe state management.

## 1. Request Handling (Goroutines)
Thunder uses Go's standard `net/http` package. Every time a user accesses the app, the server spawns a **goroutine** (a lightweight thread) specifically for that request.
- **Scalability**: Thousands of users can be handled simultaneously because goroutines are extremely cheap (starting at ~2KB of memory).
- **Isolation**: Each goroutine has its own `component.Ctx`, meaning one user's request details (params, writer, session pointer) never bleed into another's.

## 2. Thread-Safe State Management
Since multiple goroutines might try to read or write to the same `State` or `SessionStore` at the exact same time, Thunder uses **Mutual Exclusion (Mutexes)**:

### Global State (`state.State`)
Uses a `sync.RWMutex`.
- **Multiple Readers**: Many users can read the global state at once without blocking each other.
- **Exclusive Writers**: If a user updates a value, the mutex ensures that no other user is reading or writing that specific data at that exact microsecond, preventing data corruption.

### Session Store (`state.SessionStore`)
Also uses a `sync.RWMutex` to manage the map of all active sessions.
- When `getSessionState` is called, the framework safely retrieves or creates the user's private state object from the store.

## 3. "Connection Pooling"
In most web frameworks, "connection pooling" refers to database connections.
- **Current State**: Thunder currently uses an in-memory `map` for state, so there are no external database connections to "pool" yet.
- **Future Integration**: If you add a database (like PostgreSQL), Go's `database/sql` package automatically manages a **Connection Pool**. Thunder would reuse these connections across different component handlers to avoid the overhead of opening a new connection for every request.

## 4. Resource Protection (Limits)
To prevent a single user (or many users) from overwhelming the server:
- **Session Limit**: Thunder is configured with a limit of 5,000 concurrent sessions. Once reached, it stops creating new persistent sessions to protect memory.
- **Timeouts**: The server (in `server/server.go`) has built-in `ReadTimeout` and `WriteTimeout` (15s) to ensure that "hanging" connections are automatically closed, freeing up resources for other users.
