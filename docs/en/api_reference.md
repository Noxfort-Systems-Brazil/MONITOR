# Noxfort Monitor — API & Protocol Reference

This document details the ingestion protocols (MQTT & Zero-Trust HTTP), administrative REST endpoints, RBAC permissions, and Prometheus observability endpoints exposed by the Noxfort Monitor engine.

---

## 1. MQTT Telemetry Protocol

The internal Go broker client subscribes asynchronously for real-time edge ingestion.

* **Broker Address**: Configurable via `/settings` (e.g., `tcp://127.0.0.1:1883`) with live `Reconnect(brokerURL)` on configuration change and a 5-second background retry loop on startup.
* **Default Publish Topic Pattern**: `noxfort/telemetry/{identifier}`
* **Wildcard Subscription**: `noxfort/telemetry/#`
* **Strict Device Policy**: Controlled by `MONITOR_MQTT_STRICT` (default: `true`). Only pre-registered `ACTIVE` devices are accepted; messages from unregistered or `PAUSED` devices are dropped.

### Payload Format (JSON — `IncomingEvent`)

```json
{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "CRITICAL",
  "message": "Overheating detected: Temperature 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

* **`category`**: `HARDWARE` or `SOFTWARE` (routes alerts by contact role).
* **`origin`**: Unique device identifier (e.g., `pump-01`, `carina`, `synapse`).
* **`level`**: `INFO`, `WARNING`, or `CRITICAL` (`INFO` heartbeats refresh `last_seen` without alert noise; `CRITICAL` triggers synchronous `SaveEventReturningID` + alerts linked by `telemetry_id`).
* **`message`**: Human-readable event description.
* **`occurred_at`**: UTC ISO-8601 timestamp (validated within `±300s` on HTTP).

---

## 2. Zero-Trust HTTP Pairing & Telemetry Ingestion

Edge sensors operating over HTTP/HTTPS authenticate through a zero-trust out-of-band pairing flow before sending telemetry.

### Device Pre-Registration (`POST /api/pair`)

* **Authentication**: Public (unauthenticated onboarding request).
* **Size & Queue Caps**: Request body limited to 64 KB; queue capped at `50` `PENDING` devices (`429 Too Many Requests` when exceeded).
* **Payload**:

```json
{
  "origin": "edge-gw-01",
  "name": "Edge Gateway 01"
}
```

* **Responses**:
  * `202 Accepted`: Device queued in `PENDING` state. **No API key is returned in this response** (out-of-band approval: an `ADMIN` generates/copies the 5-minute TTL `nox_live_` key from the `/devices` UI).
  * `409 Conflict`: Device is already registered and `ACTIVE`.

### HTTP Telemetry Ingestion (`POST /api/telemetry`)

* **Authentication**: Required via `X-Device-Token: <api_key>` or `Authorization: Bearer <api_key>` (constant-time verification).
* **Anti-Spoofing & Replay Protection**:
  * `X-Device-Origin` header is verified against the registered device identifier/origin.
  * `occurred_at` timestamps outside a `±300s` (`±5m`) drift window are rejected.
* **Status Responses**:
  * `202 Accepted`: Buffered for asynchronous persistence via `BufferedTelemetryWriter` (and synchronous `SaveEventReturningID` on state transitions).
  * `401 Unauthorized`: Missing, expired, or invalid device token.
  * `403 Forbidden`: Device is currently `PAUSED` (Maintenance Mode) or origin mismatch.

---

## 3. Administrative REST & Web Endpoints

All administrative endpoints enforce cookie-based session authentication (`DBSessionStore`), reject `?token=` query parameters, and respect RBAC roles (`ADMIN` vs. strictly read-only `OPERATOR`).

| Endpoint | Method | Role | Description |
| :--- | :--- | :--- | :--- |
| `/api/auth/login` | `POST` | Public | Authenticates user; returns `429` lockout after 5 failed attempts in 15m (5m cooldown). |
| `/api/auth/logout` | `POST` | Operator+ | Invalidates the current session token hash in `sessions`. |
| `/api/dashboard/summary` | `GET` | Operator+ | Live JSON summary (10s polling) across all 5 states: `ONLINE`, `OFFLINE`, `PENDING`, `PAUSED`, `WAITING`. |
| `/devices/create` | `POST` | Admin | Provisions a device and displays its one-time API key. |
| `/devices/update` | `POST` | Admin | Updates/renames a registered device. |
| `/devices/toggle` | `POST` | Admin | Toggles Maintenance Mode (`PAUSED` / `ACTIVE`). |
| `/devices/rotate-key` | `POST` | Admin | Rotates the device API key and revokes the previous secret. |
| `/devices/delete` | `POST` | Admin | Permanently removes a device. |
| `/api/users` | `GET` / `POST` | Admin | Lists users or creates a new account (`ADMIN` / `OPERATOR`). |
| `/api/users/password` | `POST` | Operator+ | Self-service password change (requires `current_password`). |
| `/api/users/role` | `POST` | Admin | Updates a user's RBAC role. |
| `/api/users/unlock` | `POST` | Admin | Clears an active `LoginAttemptTracker` lockout. |
| `/api/audit/security` | `GET` | Operator+ | Queries persisted security audit logs. |
| `/api/audit/alerts` | `GET` | Operator+ | Queries notification dispatch logs (linked via `telemetry_id`). |
| `/api/audit/transitions` | `GET` | Operator+ | Queries device `ONLINE` / `OFFLINE` transition history. |
| `/api/settings/database/backups` | `GET` | Admin | Lists available SQLite/PostgreSQL hot backups. |
| `/api/settings/database/backup` | `POST` | Admin | Triggers an immediate hot backup (`VACUUM INTO` / `pg_dump -F c`). |
| `/api/settings/database/backups/download` | `GET` | Admin | Downloads a verified backup file (protected by `ResolveBackupPath`). |
| `/api/settings/database/restore` | `POST` | Admin | Restores from an uploaded or stored backup (`PRAGMA quick_check` / `pg_restore`). |

---

## 4. Health & Prometheus Observability

* **`GET /healthz`**: Lightweight liveness probe returning `200 OK` (`text/plain`).
* **`GET /api/health`**: Detailed JSON health probe reporting database status/latency, MQTT connectivity, memory, goroutines, and active device counts.
* **`GET /metrics`**: Native Prometheus exposition format reporting runtime, storage, broker, and ingestion counters:
  * `noxfort_uptime_seconds`
  * `noxfort_goroutines`
  * `noxfort_memory_heap_alloc_bytes`, `noxfort_memory_heap_sys_bytes`, `noxfort_memory_num_gc`
  * `noxfort_db_status`, `noxfort_db_latency_ms`
  * `noxfort_mqtt_connected`
  * `noxfort_devices_total`, `noxfort_devices_by_status`
  * `noxfort_telemetry_buffer_depth`
  * `noxfort_telemetry_ingested_total`, `noxfort_telemetry_rejected_total`
  * `noxfort_alerts_dispatched_total`
