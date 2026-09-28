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
| `/api/devices` | `GET` | Operator+ | Returns registered devices and `LastSeen` status. |
| `/api/devices/delete` | `POST` | Admin | Removes a device from monitoring. |
| `/api/settings/database/test` | `POST` | Admin | Tests target PostgreSQL credentials before switching. |
| `/api/settings/database/save` | `POST` | Admin | Hot-reloads active repositories with target engine (`migrate=true` supported). |
| `/api/tunnel/status` | `GET` | Authenticated | Returns state, provider (DuckDNS/Ngrok), public URLs, and IPv6. |
| `/api/tunnel/save` | `POST` | Admin | Persists DuckDNS / Ngrok parameters in database. |
| `/api/tunnel/start` | `POST` | Admin | Triggers dynamic DNS update or opens tunnel. |
| `/api/tunnel/stop` | `POST` | Admin | Stops active WAN updates and disables auto-start on boot. |
| `/api/tunnel/disconnect` | `POST` | Admin | Unlinks credentials from database and shuts down service. |
| `/api/tunnel/test` | `POST` | Admin | Actively verifies DuckDNS token and DNS resolution. |
| `/api/audit/security` | `GET` | Admin | Queries administrative security access logs. |
| `/api/audit/alerts` | `GET` | Admin | Queries alert dispatch SLA logs. |
| `/api/audit/transitions` | `GET` | Admin | Queries device failure/recovery transitions and downtimes. |
