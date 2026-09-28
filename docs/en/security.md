# 🔐 Security, Authentication & Role-Based Access Control (RBAC)

This document details the security architecture of **Noxfort Monitor™ v2.0**, including authentication, session token lifecycle, salted password hashing, and Role-Based Access Control (RBAC).

⬅️ [Central Hub](../README.md) | 🏛️ [Architecture](architecture.md) | 🔍 [Audit Trail](audit_trail.md) | 📡 [API](api_reference.md)

---

## 1. Security Architecture & RBAC

Noxfort Monitor enforces strict role segregation:

* **`RoleAdmin` ("ADMIN")**: Full administrative privileges: user management, database migration, reverse tunnel activation, and global system configuration.
* **`RoleOperator` ("OPERATOR")**: Operational privileges: viewing dashboard telemetry, monitoring registered devices, and inspecting alerts.
* **Incident Routing Roles**:
  * **`TECHNICIAN`**: Receives only `HARDWARE` incident notifications.
  * **`PROGRAMMER`**: Receives only `SOFTWARE` incident notifications.
  * **`ADMIN`**: Receives all global incidents.

---

## 2. Password Hashing & Salt Isolation

Passwords are encrypted using cryptographic salt and SHA-256 derivation via `internal/security/hasher.go`:
* Unique 16-byte random salt generated per user.
* Passwords and hashes are strictly excluded from JSON serialization (`json:"-"`).
* Constant-time comparison defends against timing attacks.

---

## 3. Session Lifecycle & AuthMiddleware

```mermaid
sequenceDiagram
    participant User as Operator / Browser
    participant MW as AuthMiddleware
    participant SM as SessionManager
    participant App as Protected Handler

    User->>MW: HTTP Request (Cookie: noxfort_session=XYZ)
    MW->>SM: ValidateSession("XYZ")
    alt Valid Session
        SM-->>MW: Session(User, RoleAdmin)
        MW->>App: Forward Request with User Context
        App-->>User: 200 OK (Render Dashboard)
    else Invalid or Expired Session
        MW-->>User: 303 Redirect to /login (or 401 Unauthorized for /api/*)
    end
```

* **Session Storage**: In-memory thread-safe map with mutex synchronization.
* **Sliding Renewal**: Active user requests renew the 24-hour expiration window.
* **Dual Interception**: Web UI requests without a session redirect to `/login` with HTTP `303 See Other`; API calls return JSON `401 Unauthorized`.
