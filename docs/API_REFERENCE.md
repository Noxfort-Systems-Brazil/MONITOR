[📚 Documentation Hub](INDEX.md) > **Complete API & Protocol Reference**

---

# 📡 Complete API & Protocol Reference: Noxfort Monitor™

This document formally specifies all communication interfaces of **Noxfort Monitor™ v2.0**, covering the asynchronous **MQTT** ingestion protocol (`noxfort/telemetry/#`), **Zero-Trust Device Pairing** (`POST /api/pair`), **HTTP Telemetry** REST ingestion (`POST /api/telemetry`), **Authentication, Sessions & Brute-Force Lockout**, **Live Dashboard**, **Device Lifecycle**, **Tunnel** control APIs, **Database & Backup/Restore** management, **Audit Trail** querying, and **Prometheus Metrics**.

---

## 1. MQTT Ingestion Protocol

The MQTT broker (Mosquitto) runs by default on `tcp://127.0.0.1:1883` (configured via `/settings` UI or `settings.mqtt_address` in the database; changing the address triggers live reconnection without restarting).

### 1.1 Publishing Topics & Zero-Trust Policy
* **Default Device Topic**: `noxfort/telemetry/{identifier}` (e.g., `noxfort/telemetry/pump-01`)
* **Server Subscription**: The Monitor Paho client subscribes using the wildcard pattern **`noxfort/telemetry/#`**, processing incoming packets non-blockingly across asynchronous goroutines.
* **MQTT Zero-Trust Policy (`MONITOR_MQTT_STRICT`)**:
  * Controlled by environment variable `MONITOR_MQTT_STRICT` (default: **`true`**; set `"false"` only in legacy development environments).
  * In strict mode, `handleMessage` verifies the payload's `origin` against `DeviceRepository`:
    * **Unregistered origin** -> Rejected (`noxfort_telemetry_rejected_total{transport="mqtt",reason="unregistered"}`).
    * **`PENDING` device** -> Rejected (`reason="pending"`).
    * **Paused device (`enabled = false`)** -> Ignored (`reason="paused"`).
  * If the broker is unreachable at boot, the client retries every **5 seconds** in the background so the HTTP server and desktop UI remain operational.

### 1.2 Universal JSON Format (`IncomingEvent`)
Both MQTT messages and HTTP REST requests must transmit the structured JSON body:

```json
{
  "category": "HARDWARE",
  "origin": "sensor-node-tx1",
  "level": "CRITICAL",
  "message": "Overheating detected: Temperature 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

#### Field Specifications:
* **`category`** (*String*, required for HTTP REST): `HARDWARE` or `SOFTWARE`. Determines RBAC alert routing rules (Hardware $\rightarrow$ Technicians; Software $\rightarrow$ Programmers).
* **`origin`** (*String*, required): Unique identifier of the edge device (e.g., `carina`, `synapse`, `pump-01`).
* **`level`** (*String*, required): Severity level: `INFO`, `WARNING`, or `CRITICAL`.
  * `INFO` messages containing keep-alive keywords ("*system ok*", "*heartbeat*", "*online*", "*ativo*", "*ok*") update the device heartbeat (`last_seen`) without creating incident records or alert noise.
  * `CRITICAL` messages trigger immediate alert dispatching to all eligible contacts.
* **`message`** (*String*, required): Human-readable operational message.
* **`occurred_at`** (*ISO-8601 Timestamp*): Precise timestamp when the event occurred at the origin (subject to ±5-minute anti-replay check on HTTP ingestion when non-zero).

### 1.3 Broker Authentication & Security
The MQTT broker enforces mandatory authentication (`allow_anonymous false`):
* **Credential Setup**: Run `make broker-auth` or `./scripts/setup_mqtt_auth.sh <user> <pass>` to configure the PBKDF2 password file (`mosquitto/config/passwd`).
* **Go Client Integration**: Reads `MQTT_USER` and `MQTT_PASSWORD` from `.env` or system environment variables, or extracts credentials embedded directly in broker connection strings (`tcp://user:pass@127.0.0.1:1883`).
* **Edge Devices (`mosquitto_pub`)**: Must specify `-u <username> -P <password>` when publishing messages.

---

## 2. Zero-Trust Device Pairing & HTTP Telemetry Ingestion

Designed for field nodes (**Carina**, **Synapse**, microcontrollers, or scripts) communicating over LAN or WAN HTTPS (Caddy/DuckDNS or Ngrok).

> [!IMPORTANT]
> **External Port Scope**: In production runtime (`ExternalIngestionHandler`), only `POST /api/pair`, `POST /api/telemetry`, `/healthz`, `/api/health`, and `/metrics` are exposed externally on port `22100`. Browser attempts to access HTML dashboard or administrative routes return **403 Forbidden**. Administrative interactions are performed within the native desktop container.

### 2.1 Step 1 — Out-of-Band Pairing (`POST /api/pair`)
Field nodes without pre-provisioned credentials request onboarding via `POST /api/pair`:

* **Body Limit & Strict Schema**: Max `64 KB`; unknown JSON fields are rejected (`DisallowUnknownFields`).
* **Rate / Anti-DoS Cap**: Maximum **50 concurrent `PENDING` devices** (`429 Too Many Requests` when exceeded).
* **Anti-Hijack Guard**: If `origin` is already `ACTIVE`, returns **`409 Conflict`** and preserves the active device key.
* **Out-of-Band Token Delivery**: Generates a 32-byte random key (`nox_live_<64 hex chars>`) with a **5-minute TTL** (300 seconds) and persists it as `PENDING`. The HTTP response **never returns the API key** — an administrator copies it from the **Monitored Systems (`/devices`)** console and injects it into the remote agent.

#### Request & Response Example:
```bash
curl -X POST http://localhost:22100/api/pair \
  -H "Content-Type: application/json" \
  -d '{"origin": "synapse-core", "name": "Synapse Perception Core"}'
```
* **`202 Accepted`**:
  ```json
  {
    "status": "pending_admin_approval",
    "message": "Pairing registered. Copy the API key generated on the Monitor console to your device settings within 5 minutes.",
    "origin": "synapse-core",
    "expires_in_seconds": 300
  }
  ```

### 2.2 Step 2 — Authenticated Telemetry (`POST /api/telemetry`)
* **Authentication**: Mandatory device API key supplied via `X-Device-Token: <api_key>` or `Authorization: Bearer <api_key>`.
  * On the **first valid transmission** of a `PENDING` device within its 5-minute TTL, `ActivateDevice` validates the token in constant time (`subtle.ConstantTimeCompare`), promotes the device to **`ACTIVE`**, clears `expires_at`, and updates `last_seen`.
  * For `ACTIVE` devices, a non-empty matching `api_key` is always enforced in constant time.
* **Anti-Spoofing Header**: Optional `X-Device-Origin` header, if provided, must strictly match JSON `origin` (`403 Forbidden` on mismatch).
* **Anti-Replay Window**: Non-zero `occurred_at` timestamps with clock drift exceeding **±5 minutes (300 seconds)** are rejected (`400 Bad Request`).
* **Content-Type & Limits**: `application/json`, max `64 KB`, strict schema (`DisallowUnknownFields`).

#### Request Example:
```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_a1b2c3d4..." \
  -H "X-Device-Origin: synapse-core" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Connection pool at 85% capacity.",
    "occurred_at": "2026-09-05T17:00:00Z"
  }'
```

#### Responses:
* **`200 OK`**: `{"status": "received"}`
* **`400 Bad Request`**: Malformed JSON, unknown fields, missing `origin`, or `occurred_at` outside ±5-minute anti-replay window.
* **`401 Unauthorized`**: Missing or invalid `X-Device-Token` / `Bearer` token.
* **`403 Forbidden`**: Unregistered device, expired `PENDING` pairing TTL, `X-Device-Origin` mismatch, or device paused in Maintenance Mode (`enabled = false`).
* **`405 Method Not Allowed`**: HTTP method other than `POST`.
* **`422 Unprocessable Entity`**: Invalid `category` (`HARDWARE`/`SOFTWARE`) or `level` (`INFO`/`WARNING`/`CRITICAL`).

---

## 3. Authentication, Sessions & Brute-Force Protection

Refer to [Security & RBAC](SECURITY.md) for architectural details.

### 3.1 `POST /api/auth/login`
Authenticates the user, tracks brute-force failures per `(username, clientIP)`, and issues a persistent session token (stored as a SHA-256 hash in `DBSessionStore`).
* **Format**: Form URL-encoded or JSON (`username`, `password`).
* **Success (`200 OK`)**: Sets `noxfort_session` cookie (`HttpOnly`, `SameSite=Lax`, `Secure` when served over HTTPS or behind a trusted TLS proxy) and returns:
  ```json
  {
    "success": true,
    "token": "<64-char-hex-token>",
    "username": "admin",
    "role": "ADMIN"
  }
  ```
* **Failure (`401 Unauthorized`)**: Includes remaining attempt telemetry when available:
  ```json
  {
    "success": false,
    "message": "Usuário ou senha incorretos",
    "failed_attempts": 2,
    "max_attempts": 5,
    "is_lockdown": false
  }
  ```
* **Brute-Force Lockout (`429 Too Many Requests`)**: Triggered after **5 failed attempts within 15 minutes** (locks for **5 minutes**):
  ```json
  {
    "success": false,
    "message": "Muitas tentativas inválidas. Tente novamente em 300 segundos.",
    "failed_attempts": 5,
    "max_attempts": 5,
    "is_lockdown": true,
    "retry_after_seconds": 300
  }
  ```

### 3.2 `POST /api/auth/logout`
Invalidates the token in [`DBSessionStore`](../internal/storage/session_repo.go) and expires the cookie (`MaxAge: -1`).

### 3.3 `GET /api/auth/status`
Returns the authentication state of the current session (extracted from cookie `noxfort_session`, `Authorization: Bearer`, or `X-Session-Token`; URL query `?token=` is intentionally rejected).
```json
{
  "authenticated": true,
  "username": "admin",
  "role": "ADMIN"
}
```

### 3.4 `POST /api/auth/register`
Protected registration endpoint (`RoleAdmin` required). Functions as a secure alias for `/api/users/create` and enforces [`ValidatePasswordComplexity`](../internal/security/password_validator.go) (minimum 8 characters, digit, uppercase, lowercase, special character). Returns `400 Bad Request` if criteria are unmet.

---

## 4. Dashboard & Operator Account Management

### 4.1 Live Dashboard (`/`, `/api/dashboard/summary`)
* `GET /`: Renders the main System Overview HTML dashboard (`Operator+`).
* `GET /api/dashboard/summary` (`Operator+`): Returns live JSON snapshot polled every 10 seconds by the UI, including 5-state lifecycle classification (`ONLINE`, `OFFLINE`, `PENDING`, `PAUSED`, `WAITING`) and recent incidents:
  ```json
  {
    "now": "2026-11-16T14:00:00-03:00",
    "devices": [
      {
        "id": 1,
        "name": "Synapse Primary",
        "identifier": "synapse-main",
        "last_seen": "13:59:52 16/11/2026",
        "status": "ONLINE",
        "badge_class": "bg-success",
        "row_class": "table-success"
      }
    ],
    "recent_incidents": []
  }
  ```

### 4.2 Account Management (`/users`, `/api/users`)
*Note: Operators operate strictly in read-only mode across the application. All endpoints below except self-service password change require `ADMIN`.*

| Method | Route | Role | Description | Parameters / Body |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/users` | Admin | Renders user account management interface (`403` for Operators). | - |
| `GET` | `/api/users` | Admin | Lists all registered accounts (hashes omitted via `json:"-"`). | - |
| `POST` | `/api/users/create` | Admin | Creates a new operator or administrator (enforces password complexity). | `username`, `password`, `role` (`ADMIN` or `OPERATOR`) |
| `POST` | `/api/users/password` | Authenticated | Self-service password change (any role, requires `current_password`) or admin reset. Revokes active sessions and clears lockout. | `username` (optional for self), `current_password` (non-admin self), `new_password` |
| `POST` | `/api/users/role` | Admin | Promotes/demotes between `ADMIN` and `OPERATOR` (protects superuser and last `ADMIN`); revokes target sessions. | `username`, `role` (`ADMIN` \| `OPERATOR`) |
| `POST` | `/api/users/unlock` | Admin | Clears brute-force lockout counters for a locked username across all IPs. | `username` |
| `POST` | `/api/users/delete` | Admin | Deletes a user account (protects superuser and last `ADMIN`); revokes active sessions. | `username` |

---

## 5. Response Team & Incident Contacts (`/contacts`)

| Method | Route | Role | Description | Parameters / Body |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/contacts` | Operator+ | Renders the Response Team directory (read-only view for Operators). | - |
| `POST` | `/contacts/create` | Admin | Adds a new alert recipient and logs `CONTACT_CREATED`. | `name`, `email`, `phone`, `role` (`TECHNICIAN`, `PROGRAMMER`, `ADMIN`), `notify_critical` (`on`), `telegram_chat_id` |
| `POST` | `/contacts/update` | Admin | Updates an existing contact record and logs `CONTACT_UPDATED`. | `id` (int), `name`, `email`, `phone`, `role`, `notify_critical`, `telegram_chat_id` |
| `POST` | `/contacts/delete` | Admin | Removes an alert recipient and logs `CONTACT_DELETED`. | `id` (int) |

---

## 6. Monitored Equipment & System Registry (`/devices`)

| Method | Route | Role | Description | Parameters / Query |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/devices` | Operator+ | Renders active and `PENDING` devices. Active `api_key` values are redacted from HTML; `PENDING` keys are shown for 5m onboarding copy. | - |
| `POST` | `/devices/create` | Admin | Manually provisions an `ACTIVE` device and returns a one-time `nox_live_` API key (`JSON` if `Accept: application/json`, else `303`). | `identifier`, `name` (optional) |
| `POST` | `/devices/update` | Admin | Renames an existing monitored device (`DEVICE_UPDATED`). | `identifier` or `id`, `name` |
| `POST` | `/devices/toggle` | Admin | Pauses or resumes monitoring/alerts (Maintenance Mode: `DEVICE_PAUSED` / `DEVICE_RESUMED`). | `identifier` or `id`, `enabled` (`true` \| `1` \| `on` \| `false`) |
| `POST` | `/devices/rotate-key` | Admin | Issues a new `nox_live_` API key for an `ACTIVE` device (`DEVICE_KEY_ROTATED`) and returns it once in JSON. | `identifier` or `id` |
| `POST` | `/devices/delete` | Admin | Removes a device from monitoring or cancels a pending pairing request (`DEVICE_DELETE`). | `id` (form-encoded body parameter) |

---

## 7. Remote Access & WAN Tunnel API (`/api/tunnel`)

Refer to [Remote Access & WAN Ingestion](REMOTE_ACCESS.md).

### 7.1 `GET /api/tunnel/status`
Returns real-time WAN ingestion status, active provider, public endpoints, and connection parameters (`Operator+`):
```json
{
  "state": "ONLINE",
  "provider": "DuckDNS",
  "public_url": "https://noxfort-monitor.duckdns.org",
  "telemetry_url": "https://noxfort-monitor.duckdns.org/api/telemetry",
  "domain": "noxfort-monitor",
  "binary_found": true,
  "ipv6_address": "2804:14d:...",
  "local_port": "22100",
  "use_https": true,
  "error_message": "",
  "started_at": "01:30:00 28/09/2026"
}
```

### 7.2 Tunnel Management Endpoints:
* `GET /remote` (`Operator+`): Renders Remote Access UI (redacts saved `duckdns_token` from HTML).
* `POST /api/tunnel/save` (`Admin`): Persists WAN configuration (`duckdns_token`, `duckdns_domain`, `duckdns_enabled`); preserves stored token if blank during domain-only updates.
* `POST /api/tunnel/start` (`Admin`): Triggers dynamic DNS update or opens tunnel on demand (`TUNNEL_STARTED`).
* `POST /api/tunnel/stop` (`Admin`): Terminates the active tunnel or pauses DuckDNS auto-updates on boot (`TUNNEL_STOPPED`).
* `POST /api/tunnel/disconnect` (`Admin`): Clears stored WAN credentials from the database and terminates service (`TUNNEL_DISCONNECTED`).
* `POST /api/tunnel/test` (`Admin`): Validates DuckDNS credentials and DNS resolution actively without modifying settings.

---

## 8. Database, Backup & Restore API (`/api/settings/database`)

Refer to [Database & Dual-Engine Persistence](DATABASE.md). Note that `/api/settings/database/backup*` routes are strictly restricted to **`ADMIN`** (even on `GET` requests) because snapshots contain password hashes and device API keys.

* `GET /server` (`Operator+`): Renders Dual-Engine Storage & Backup UI (backup controls visible only to `ADMIN`).
* `GET /api/settings/database/status` (`Operator+`): Returns active database engine status, ping latency, and sanitized configuration (password omitted).
* `POST /api/settings/database/test` (`Admin`): Tests target PostgreSQL/SQLite connectivity without altering the active connection.
* `POST /api/settings/database/save` (`Admin`): Hot-reloads active repositories (`migrate=true` runs single-transaction `MigrateData` and aborts switch on failure).
* `POST /api/settings/database/provision-user` (`Admin`): Provisions a dedicated schema-restricted PostgreSQL user.
* `POST /api/settings/database/backup` (`Admin`): Triggers a non-blocking hot database snapshot (`VACUUM INTO` / `pg_dump -F c`), keeping up to 7 files.
* `GET /api/settings/database/backups` (`Admin`): Lists available snapshots sorted descending by timestamp.
* `GET /api/settings/database/backups/download?file=<filename>` (`Admin`): Validates filename against path traversal (`ResolveBackupPath`), logs `DATABASE_BACKUP_DOWNLOADED`, and streams the snapshot file.
* `POST /api/settings/database/restore` (`Admin`): Accepts `filename` (JSON or form-encoded), validates SQLite snapshots via `PRAGMA quick_check;` (or invokes `pg_restore --clean --if-exists` for `.sql`), atomically swaps the live SQLite file and hot-reloads all repositories, logging `DATABASE_BACKUP_RESTORED`.

---

## 9. Audit Trail (`/api/audit`)

Refer to [Audit Trail](AUDIT_TRAIL.md). All audit endpoints are read-only (`Operator+`):
* `GET /audit`: Renders the triple audit trail HTML view.
* `GET /api/audit/security`: Returns the latest 100 security event logs.
* `GET /api/audit/alerts`: Returns the latest 100 alert dispatch logs (including linked `telemetry_id`).
* `GET /api/audit/transitions`: Returns the latest 100 Watchdog device outage and recovery transitions.

---

## 10. System Settings & Notification Channel Diagnostics

* `GET /settings` (`Operator+`): Renders SMTP, Telegram, and MQTT settings (redacting stored `SMTPPass` and `TelegramBotToken` from the DOM).
* `POST /settings/save` (`Admin`): Section-aware partial merge via `form_section` (`smtp`, `telegram`, `mqtt`) preserving untouched secrets; changing `mqtt_address` triggers immediate live MQTT client reconnection (`Reconnect`).
* `POST /settings/test` (`Admin`): Sends a test email using the active SMTP settings (supports implicit TLS on port 465 and STARTTLS).
* `POST /settings/test-telegram` (`Admin`): Sends a MarkdownV2 test message to the specified `chat_id`.

---

## 11. Desktop and Browser Controls

* `POST /api/open-external` (`Public`/`Desktop`): Accepts `url` (`http://` or `https://` only) and launches the host's default browser (`xdg-open`, `open`, `rundll32`).
* `POST /api/window/toggle-fullscreen` (`Public`): Toggles the Wails window between fullscreen and normal mode.
* `POST /api/window/exit-fullscreen` (`Public`): Exits fullscreen mode.

---

## 12. Observability & Application Diagnostics

Public endpoints exempt from `AuthMiddleware` and exposed on both `ExternalIngestionHandler` and `DesktopHandler`:

### 12.1 `GET /healthz`
Liveness probe returning `200 OK` (`"OK\n"`).

### 12.2 `GET /api/health`
Deep diagnostic health JSON (`200 OK` when `healthy`/`degraded`, `503 Service Unavailable` when `unhealthy` if active DB is disconnected).

### 12.3 `GET /metrics`
Prometheus exposition format (`text/plain; version=0.0.4; charset=utf-8`) exporting:
* **Runtime & Connectivity Gauges/Counters**:
  * `noxfort_uptime_seconds` (gauge)
  * `noxfort_goroutines` (gauge)
  * `noxfort_memory_heap_alloc_bytes` (gauge)
  * `noxfort_memory_heap_sys_bytes` (gauge)
  * `noxfort_memory_num_gc` (counter)
  * `noxfort_db_status{type="sqlite|postgres"}` (gauge)
  * `noxfort_db_latency_ms` (gauge)
  * `noxfort_mqtt_connected` (gauge)
* **Device & Ingestion Instrumentation (`internal/metrics`)**:
  * `noxfort_devices_total` (gauge)
  * `noxfort_devices_by_status{state="online|offline|pending|paused|waiting"}` (gauge)
  * `noxfort_telemetry_buffer_depth` (gauge)
  * `noxfort_telemetry_ingested_total{transport="http|mqtt",level="INFO|WARNING|CRITICAL"}` (counter)
  * `noxfort_telemetry_rejected_total{transport="http|mqtt",reason="..."}` (counter)
  * `noxfort_alerts_dispatched_total{channel="email|telegram",status="SENT|FAILED"}` (counter)

---

### 🔗 Related Documentation
* 🏗️ [System Architecture](../ARCHITECTURE.md) — Data flow overview
* 🗄️ [Database & Persistence](DATABASE.md) — Models, WAL mode, backup/restore & retention
* 🔐 [Security & RBAC](SECURITY.md) — Zero-Trust pairing, lockout & strict Operator RBAC
* 🌐 [Remote Access](REMOTE_ACCESS.md) — DuckDNS & Ngrok tunnel configuration
* 🔍 [Audit Trail](AUDIT_TRAIL.md) — Detailed log schema
* 🧪 [Testing Guide](TESTING.md) — Practical testing examples with cURL and mosquitto
