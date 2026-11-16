# Noxfort Monitor — Security Architecture

Noxfort Monitor implements defense-in-depth across user authentication, session persistence, edge device onboarding, secret handling, and alert transport.

---

## 1. Role-Based Access Control (RBAC)

Two operational roles govern access across both the HTTP API and the embedded Wails desktop UI:

* **`ADMIN`**: Full administrative privileges across device provisioning/key rotation, user/role/unlock management, alert contacts, system/MQTT settings, DuckDNS/Ngrok remote tunnels, and database switching/backup/restore. Superuser and last-admin accounts are protected against deletion (`DeleteUserByActor`).
* **`OPERATOR` (Strictly Read-Only)**: Restricted to read-only observation (`GET` dashboard, `/audit`, `/api/audit/*`, `/remote`, `/server`, `/settings`), `/api/auth/logout`, and self-service `POST /api/users/password` (with `current_password`). Blocked from `/users`, `/api/users` mutations, `/api/settings/database/backup*`, and all state-changing actions:
  * API / XHR clients (`Accept: application/json`, `/api/*`, `XMLHttpRequest`) receive `403 Forbidden` JSON.
  * Desktop/browser HTML views render the custom `web/templates/forbidden.html` error page.

---

## 2. Password Hashing & Complexity Policy

* **Derivation Algorithm**: Passwords are hashed by `PBKDF2Hasher` using **PBKDF2 HMAC-SHA256** (`100,000` iterations, `16`-byte cryptographically random salt, `32`-byte derived key).
* **Complexity Enforcement**: `ValidatePasswordComplexity` enforces length and character diversity requirements across account creation, password changes, and superuser initialization.

---

## 3. Brute-Force Protection (`LoginAttemptTracker`)

* Failed authentication attempts are tracked per composite key (`username|ip`, resolved through `ExtractClientIP`).
* Threshold: **`5` failed attempts** within a **`15-minute`** sliding window triggers a **`5-minute` lockout**, returning HTTP **`429 Too Many Requests`** and logging `AUTH_LOGIN_LOCKED`.
* Administrators can manually inspect and lift lockouts via `POST /api/users/unlock`.

---

## 4. Persistent Hashed Sessions & Proxy Awareness

* **Persistent Store (`DBSessionStore`)**: Active sessions are persisted in the `sessions` table rather than ephemeral memory. Only the **`SHA-256` hash (`token_hash`)** of the bearer token is stored, with a fixed **`24-hour` TTL** and an hourly background cleanup sweep. Because `DBSessionStore` implements `ReloadableRepository`, sessions survive live SQLite/PostgreSQL engine switches.
* **No Query Tokens**: `ExtractSessionToken` reads the HTTP-only cookie (`Set-Cookie`) and explicitly rejects tokens passed in `?token=` URL query strings to prevent log and referrer leakage.
* **Trusted Proxy & HTTPS Detection**: `ExtractClientIP` and `IsHTTPSRequest` evaluate `X-Real-IP`, `X-Forwarded-For`, and `X-Forwarded-Proto` (forwarded by `caddy/Caddyfile`) only against CIDR/IP entries in `TRUSTED_PROXIES`, combined with `MONITOR_USE_HTTPS`, so `Secure` cookies and audit IPs remain tamper-resistant.

---

## 5. Edge Ingestion & Alert Transport Hardening

* **Zero-Trust Device Pairing**: `POST /api/pair` registers untrusted devices as `PENDING` (Queue limit: 50, `409 Conflict` if already `ACTIVE`) and returns `202 Accepted` without ever returning a secret. The 5-minute TTL `nox_live_` token is issued out-of-band from the `/devices` UI and validated in constant time on `POST /api/telemetry` alongside `X-Device-Origin` and `±300s` replay protection.
* **Strict MQTT Ingestion**: `MONITOR_MQTT_STRICT` (default `true`) drops MQTT payloads (`noxfort/telemetry/#`) from unknown or `PAUSED` devices.
* **HTML Secret Redaction**: Web views (`/settings`, `/remote`, `/devices`) mask active `ApiKey` secrets and tokens such as `duckdns_token` in rendered HTML; partial configuration saves preserve unedited credentials.
* **SMTP Hardening**: `EmailChannel` supports implicit TLS on port `465` (`TLS 1.2+` minimum) and applies `sanitizeEmailHeader` (CRLF stripping) to block SMTP header injection.
