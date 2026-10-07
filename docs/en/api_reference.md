# 📡 API Reference & Ingestion Protocols

This document specifies the communication interfaces of **Noxfort Monitor™ v2.0**, including asynchronous MQTT ingestion, HTTP REST telemetry, and administrative management endpoints.

⬅️ [Central Hub](../README.md) | 🏛️ [Architecture](architecture.md) | 🗄️ [Database](database.md) | 🔐 [Security](security.md)

---

## 1. MQTT Ingestion Protocol

* **Broker Address**: `tcp://127.0.0.1:1883`
* **Default Topic Pattern**: `noxfort/devices/{identifier}/telemetry`
* **Wildcard Subscription**: `noxfort/devices/+/telemetry`

### Universal `IncomingEvent` JSON Payload

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
* **`category`** (*String*): `HARDWARE` or `SOFTWARE`. Controls RBAC routing (Hardware $\rightarrow$ Technicians; Software $\rightarrow$ Programmers).
* **`origin`** (*String*): Unique device identifier (e.g. `carina`, `synapse`, `pump-01`).
* **`level`** (*String*): `INFO`, `WARNING`, or `CRITICAL`.
  * `INFO` with keep-alive keywords ("*system ok*", "*heartbeat*", "*online*") updates the heartbeat timestamp without creating alert noise or redundant DB writes.
  * `CRITICAL` triggers instant multi-channel alert dispatching.
* **`message`** (*String*): Human-readable operational description.
* **`occurred_at`** (*ISO-8601 String*): UTC timestamp.

---

## 2. Telemetry Ingestion via HTTP REST

### `POST /api/telemetry`
Designed for edge agents (**Carina**, **Synapse**, IoT microcontrollers, or cURL scripts).

* **Port**: `22100` (or public HTTPS URL via DuckDNS/Caddy or Ngrok)
* **Authentication**: Public (exempt from middleware for autonomous sensor nodes).
* **Network Isolation**: In production (`ExternalIngestionHandler`), `POST /api/telemetry` is the **only accessible external endpoint**. Direct browser requests to dashboard routes return **403 Forbidden**.
* **Header**: `Content-Type: application/json`
* **Body**: Identical to `IncomingEvent` JSON schema.

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Connection pool at 85% capacity.",
    "occurred_at": "2026-09-27T18:00:00Z"
  }'
```

---

## 3. Administrative & Operations API

*Available inside the native desktop application or over authenticated sessions.*

| Endpoint | Method | Role | Description |
|---|---|---|---|
| `/api/auth/login` | `POST` | Public | Authenticates credentials and issues `noxfort_session` cookie. |
| `/api/auth/logout` | `POST` | Authenticated | Invalidates the active session token. |
| `/api/auth/status` | `GET` | Public | Returns session authentication state, username, and role. |
| `/api/auth/register` | `POST` | Admin | Protected account creation alias enforcing industrial password complexity. |
| `/devices` | `GET` | Operator+ | Renders system management dashboard with registered nodes and `LastSeen` status. |
| `/devices/delete` | `GET` | Admin | Removes a device via query parameter (`?id=xyz`) and redirects (`303`). |
| `/contacts` | `GET` | Operator+ | Renders the Response Team management interface. |
| `/contacts/create` | `POST` | Admin | Registers a new alert recipient (name, email, phone, telegram_chat_id, role). |
| `/contacts/update` | `POST` | Admin | Updates contact information and notification channels. |
| `/contacts/delete` | `POST` | Admin | Removes an alert recipient from the response team. |
| `/api/users` | `GET` | Admin | Returns JSON array of all registered operators. |
| `/api/users/create` | `POST` | Admin | Provisions operator/admin account (enforces password complexity). |
| `/api/users/delete` | `POST` | Admin | Deletes an operator account (superuser cannot be removed). |
| `/api/settings/database/status` | `GET` | Authenticated | Returns active database engine, latency, and schema status. |
| `/api/settings/database/test` | `POST` | Admin | Tests target PostgreSQL credentials before switching. |
| `/api/settings/database/save` | `POST` | Admin | Hot-reloads active repositories with target engine (`migrate=true` supported). |
| `/api/settings/database/provision-user` | `POST` | Admin | Provisions a schema-restricted dedicated user in PostgreSQL. |
| `/api/settings/database/backup` | `POST` | Admin | Triggers a non-blocking hot database snapshot (`VACUUM INTO` / `pg_dump`). |
| `/api/settings/database/backups` | `GET` | Admin | Lists all available backup snapshots in `backups/`. |
| `/api/tunnel/status` | `GET` | Authenticated | Returns state, provider (DuckDNS/Ngrok), public URLs, and IPv6. |
| `/api/tunnel/save` | `POST` | Admin | Persists DuckDNS / Ngrok parameters in database. |
| `/api/tunnel/start` | `POST` | Admin | Triggers dynamic DNS update or opens tunnel. |
| `/api/tunnel/stop` | `POST` | Admin | Stops active WAN updates and disables auto-start on boot. |
| `/api/tunnel/disconnect` | `POST` | Admin | Unlinks credentials from database and shuts down service. |
| `/api/tunnel/test` | `POST` | Admin | Actively verifies DuckDNS token and DNS resolution. |
| `/api/audit/security` | `GET` | Admin | Queries administrative security access logs. |
| `/api/audit/alerts` | `GET` | Admin | Queries alert dispatch SLA logs. |
| `/api/audit/transitions` | `GET` | Admin | Queries device failure/recovery transitions and downtimes. |
| `/settings/test` | `POST` | Admin | Sends a test email using active SMTP configuration. |
| `/settings/test-telegram` | `POST` | Admin | Sends a MarkdownV2 test alert to configured Telegram bot and chat ID. |
| `/api/open-external` | `POST` | Authenticated | Directs operating system to open target URL in default browser. |
| `/api/window/toggle-fullscreen` | `POST` | Public | Toggles Wails window between fullscreen and normal mode. |
| `/api/window/exit-fullscreen` | `POST` | Public | Exits fullscreen mode. |

---

## 4. Public Observability & Application Diagnostics

*Exempt from authentication middleware to allow container health checks and Prometheus scraping.*

| Endpoint | Method | Description | Format |
|---|---|---|---|
| `/healthz` | `GET` | Liveness probe returning `200 OK` (`"OK\n"`). | `text/plain` |
| `/api/health` | `GET` | Deep diagnostic probe: database status & latency, MQTT broker connectivity, RAM allocation, goroutines, and device count. | `application/json` |
| `/metrics` | `GET` | **Prometheus** metrics export (`noxfort_uptime_seconds`, `noxfort_goroutines`, `noxfort_db_status`, `noxfort_mqtt_connected`). | `text/plain; version=0.0.4` |
