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
Thunder provides built-in protection via cookie policies:
- **SameSite=Lax**: By default, session cookies use the `SameSite=Lax` flag. This prevents the browser from sending the session cookie during cross-site `POST` requests, which is the primary vector for CSRF attacks.
- **Action Pattern**: The `Action` system in Thunder encourages the use of `POST` for state changes, combined with the `SameSite` policy, providing a layer of defense-in-depth.

## 4. Session Security
- **Secure ID Generation**: Session IDs are 16-byte cryptographically secure random values (32-character hex strings), making them virtually impossible to guess or brute-force.
- **Session Expiration**: Automatic background cleanup ensures that stale sessions are pruned, reducing the window of opportunity for session hijacking.
- **HTTPS Enforcement**: The `Secure` flag is automatically applied to cookies if the application is accessed over a TLS/SSL connection.
