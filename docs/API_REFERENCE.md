[📚 Documentation Hub](INDEX.md) > **Complete API & Protocol Reference**

---

# 📡 Complete API & Protocol Reference: Noxfort Monitor™

This document formally specifies all communication interfaces of **Noxfort Monitor™ v2.0**, covering the asynchronous **MQTT** ingestion protocol, the **HTTP Telemetry** REST endpoint, the **Authentication & Sessions** subsystem, **Ngrok Tunnel** control APIs, **Database** management, and **Audit Trail** querying.

---

## 1. MQTT Ingestion Protocol

The MQTT broker (Mosquitto) runs by default on `tcp://127.0.0.1:1883` (configurable in `Settings` or `configs/config.yaml`).

### 1.1 Publishing Topics
* **Default Device Topic**: `noxfort/devices/{identifier}/telemetry`
* **Server Subscription**: The Monitor client subscribes using wildcards (`noxfort/devices/+/telemetry` or dedicated topics), processing incoming packets non-blockingly across asynchronous goroutines.

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
* **`category`** (*String*, required): `HARDWARE` or `SOFTWARE`. Determines RBAC alert routing rules (Hardware $\rightarrow$ Technicians; Software $\rightarrow$ Programmers).
* **`origin`** (*String*, required): Unique identifier of the edge device (e.g., `carina`, `synapse`, `pump-01`).
* **`level`** (*String*, required): Severity level: `INFO`, `WARNING`, or `CRITICAL`.
  * `INFO` messages containing keep-alive keywords ("*system ok*", "*heartbeat*", "*online*") update the device heartbeat (`last_seen`) but are omitted from permanent persistence to optimize storage.
  * `CRITICAL` messages trigger immediate alert dispatching to all eligible contacts.
* **`message`** (*String*, required): Human-readable operational message.
* **`occurred_at`** (*ISO-8601 Timestamp*, required): Precise timestamp when the event occurred at the origin.

### 1.3 Broker Authentication & Security
The MQTT broker enforces mandatory authentication (`allow_anonymous false`):
* **Credential Setup**: Run `make broker-auth` or `./scripts/setup_mqtt_auth.sh <user> <pass>` to configure the PBKDF2 password file (`mosquitto/config/passwd`).
* **Go Client Integration**: Reads `MQTT_USER` and `MQTT_PASSWORD` from `.env` or system environment variables, or extracts credentials embedded directly in broker connection strings (`tcp://user:pass@127.0.0.1:1883`).
* **Edge Devices (`mosquitto_pub`)**: Must specify `-u <username> -P <password>` when publishing messages.

---

## 2. Telemetry Ingestion via HTTP REST

Designed for field nodes (such as **Carina**, **Synapse**, or cURL/Python scripts) operating in external networks where direct MQTT connections to port 1883 are blocked.

### `POST /api/telemetry`
* **Authentication**: Public (exempt from auth middleware to accommodate autonomous field sensors).
* **Content-Type**: `application/json`
* **Body**: Identical to the universal `IncomingEvent` JSON structure.

> [!IMPORTANT]
> **External Port Scope**: In production runtime (`ExternalIngestionHandler`), `POST /api/telemetry` is the **only route** exposed externally on port `22100`. Browser attempts to access HTML dashboard or administrative routes return **403 Forbidden**. Administrative interactions are performed within the native desktop container.

#### Request Example:
```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Connection pool at 85% capacity.",
    "occurred_at": "2026-09-05T17:00:00Z"
  }'
```

#### Responses:
* **`200 OK`**:
  ```json
  {"status": "received"}
  ```
* **`400 Bad Request`**: Malformed body or missing required fields (`origin`, `level`, or `message`).
* **`405 Method Not Allowed`**: When invoked with HTTP methods other than `POST`.

---

## 3. Authentication & Session Management

Refer to [Security & RBAC](SECURITY.md) for architectural details.

### 3.1 `POST /api/auth/login`
Authenticates the user and issues a session cookie.
* **Format**: Form URL-encoded or JSON (`username`, `password`).
* **Success (`200 OK`)**: Sets the `noxfort_session` cookie with security flags.
  ```json
  {"success": true, "redirect": "/"}
  ```
* **Failure (`401 Unauthorized`)**:
  ```json
  {"success": false, "error": "Invalid credentials"}
  ```

### 3.2 `POST /api/auth/logout`
Invalidates the token in the [`SessionManager`](../internal/security/session.go) and expires the cookie in the client/browser.

### 3.3 `GET /api/auth/status`
Returns the authentication state of the current session.
* **Authenticated Response**:
  ```json
  {
    "authenticated": true,
    "username": "admin",
    "role": "ADMIN"
  }
  ```

---

## 4. Operator Account Management (`/api/users`)

*Requires an authenticated session with `ADMIN` role.*

| Method | Route | Description | Parameters / Body |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/users` | Lists all registered operators. | - |
| `POST` | `/api/users/create` | Creates a new operator or administrator. | `username`, `password`, `role` (`ADMIN` or `OPERATOR`) |
| `POST` | `/api/users/delete` | Deletes the specified user account. | `username` (Query or Form) |

---

## 5. Remote Access & WAN Tunnel API (`/api/tunnel`)

Refer to [Remote Access & WAN Ingestion](REMOTE_ACCESS.md) for conceptual and setup details.

### 5.1 `GET /api/tunnel/status`
Returns real-time WAN ingestion status, active provider, public endpoints, and connection parameters:
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

### 5.2 Additional Tunnel Endpoints:
* `POST /api/tunnel/save`: Persists WAN configuration. Supports DuckDNS (`duckdns_token`, `duckdns_domain`, `duckdns_enabled`) and Ngrok parameters.
* `POST /api/tunnel/start`: Triggers dynamic DNS update or opens reverse tunnel on demand.
* `POST /api/tunnel/stop`: Terminates the active tunnel or pauses DuckDNS auto-updates on boot.
* `POST /api/tunnel/disconnect`: Clears stored WAN credentials from the database and terminates service.
* `POST /api/tunnel/test`: Validates DuckDNS credentials and DNS resolution actively without modifying settings.
  * **Payload Response (`200 OK`)**:
    ```json
    {
      "success": true,
      "message": "Conexão DuckDNS validada com sucesso!",
      "domain": "noxfort-monitor.duckdns.org",
      "resolved_ips": ["177.136.20.10"],
      "ipv6_active": true
    }
    ```

---

## 6. Database & Server Config (`/api/settings/database`)

Refer to [Database & Dual-Engine Persistence](DATABASE.md).

### 6.1 `GET /api/settings/database/status`
Returns the active database engine and query latency:
```json
{
  "connected": true,
  "type": "postgres",
  "host": "localhost",
  "port": 5432,
  "dbname": "noxfort_database",
  "schema": "schema_monitor",
  "user": "user_monitor",
  "latency_ms": 2,
  "schema_exists": true,
  "version": "PostgreSQL 16.1",
  "server_time": "2026-09-05T17:05:00Z"
}
```

### 6.2 `POST /api/settings/database/test`
Tests connectivity with supplied parameters without altering the active production database connection.

### 6.3 `POST /api/settings/database/save`
Applies new credentials to the [`DBManager`](../internal/storage/db_manager.go). If `migrate=true` is passed, runs data synchronization prior to switching the driver.

### 6.4 `POST /api/settings/database/provision-user`
Provisions a dedicated schema-restricted user using PostgreSQL administrative credentials.

### 6.5 `POST /api/settings/database/backup`
Triggers an immediate, non-blocking hot database backup:
* **SQLite**: Executes `VACUUM INTO` to produce a consistent binary `.db` snapshot in `backups/`.
* **PostgreSQL**: Invokes `pg_dump` with custom compression.
* **Rotation**: Automatically retains the latest 7 daily snapshots.
* **Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "message": "Backup gerado com sucesso!",
    "backup": {
      "filename": "monitor_sqlite_20260928_022930.db",
      "file_path": "/home/user/.../backups/monitor_sqlite_20260928_022930.db",
      "size_bytes": 4096,
      "driver": "sqlite",
      "created_at": "2026-09-28T02:29:30Z"
    }
  }
  ```

### 6.6 `GET /api/settings/database/backups`
Returns an array of all available backup files sorted by creation date descending.

---

## 7. Audit Trail (`/api/audit`)

Refer to [Audit Trail](AUDIT_TRAIL.md).

* `GET /api/audit/security?limit=100`: Security event log history.
* `GET /api/audit/alerts?limit=100`: Alert dispatch history via Email/Telegram with `SENT`/`FAILED` status.
* `GET /api/audit/transitions?limit=100`: Device outage and recovery history recorded by the Watchdog Engine with downtime durations.

---

## 8. Notification Channel Diagnostics

Endpoints used to dispatch verification alerts on demand:
* `POST /settings/test`: Sends a test email using the active SMTP settings.
* `POST /settings/test-telegram`: Sends a MarkdownV2 test message using the configured bot token and chat ID.

---

## 9. Desktop and Browser Controls

* `POST /api/open-external`: Accepts `{"url": "https://..."}` and directs the operating system to open the link in the user's default browser (`xdg-open` on Linux).
* `POST /api/window/toggle-fullscreen`: Toggles the Wails window between fullscreen and normal mode.
* `POST /api/window/exit-fullscreen`: Exits fullscreen mode.

---

## 10. Observability & Application Diagnostics

Noxfort Monitor provides public diagnostic and metrics endpoints for infrastructure monitoring, container health probes, and Prometheus scrapers (exempt from authentication middleware):

### 10.1 `GET /healthz`
Liveness probe endpoint. Responds with `200 OK` and payload `"OK\n"` when the HTTP server is alive and accepting traffic.

### 10.2 `GET /api/health`
Deep diagnostic health endpoint returning component connectivity, memory statistics, and process metadata.
* **Success (`200 OK`)**: System is `healthy` or `degraded`.
* **Service Unavailable (`503 Service Unavailable`)**: Critical components (active database) are offline.
* **Payload Example**:
  ```json
  {
    "status": "healthy",
    "timestamp": "2026-09-28T02:29:30Z",
    "uptime_seconds": 1240,
    "components": {
      "database": {
        "status": "up",
        "type": "sqlite",
        "latency_ms": 2,
        "error": ""
      },
      "mqtt": {
        "connected": true,
        "status": "up"
      },
      "devices": {
        "count": 14
      }
    },
    "system": {
      "goroutines": 28,
      "heap_alloc_mb": 18,
      "heap_sys_mb": 34,
      "num_gc": 4,
      "go_version": "go1.22.2"
    }
  }
  ```

### 10.3 `GET /metrics`
Prometheus metrics exporter endpoint in text exposition format (`version 0.0.4`):
```promql
# HELP noxfort_uptime_seconds Application uptime in seconds.
# TYPE noxfort_uptime_seconds gauge
noxfort_uptime_seconds 1240

# HELP noxfort_goroutines Current number of running goroutines.
# TYPE noxfort_goroutines gauge
noxfort_goroutines 28

# HELP noxfort_db_status Active database connection status (1 = connected, 0 = disconnected).
# TYPE noxfort_db_status gauge
noxfort_db_status{type="sqlite"} 1

# HELP noxfort_mqtt_connected MQTT broker connection status (1 = connected, 0 = disconnected).
# TYPE noxfort_mqtt_connected gauge
noxfort_mqtt_connected 1
```

---

### 🔗 Related Documentation
* 🏗️ [System Architecture](../ARCHITECTURE.md) — Data flow overview
* 🗄️ [Database & Persistence](DATABASE.md) — Models and DDL schemas
* 🔐 [Security & RBAC](SECURITY.md) — Authentication mechanisms
* 🌐 [Remote Access](REMOTE_ACCESS.md) — Ngrok tunnel configuration
* 🔍 [Audit Trail](AUDIT_TRAIL.md) — Detailed log schema
* 🧪 [Testing Guide](TESTING.md) — Practical testing examples with cURL and mosquitto
