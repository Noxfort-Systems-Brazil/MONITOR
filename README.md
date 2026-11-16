<div align="center">

<img src="docs/assets/monitor-logo.png" alt="Noxfort Monitor Logo" width="130" />

# NOXFORT MONITOR™
### Industrial Telemetry Ingestion, Observability & Incident Response Orchestration
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Active-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![License: AGPL-3.0-or-later](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](LICENSE)

[![Noxfort Monitor GitHub Repository Card](https://github-readme-stats.vercel.app/api/pin/?username=Noxfort-Systems-Brazil&repo=MONITOR&theme=dark)](https://github.com/Noxfort-Systems-Brazil/MONITOR)

---

🌐 **Translations / Idiomas:** **[🇺🇸 English](README.md)** • **[🇧🇷 Português do Brasil](docs/pt-br/README.md)** • **[🇪🇸 Español](docs/es/README.md)** • **[🇫🇷 Français](docs/fr/README.md)** • **[🇷🇺 Русский](docs/ru/README.md)** • **[🇨🇳 简体中文](docs/zh/README.md)** • **[📚 Documentation Hub](docs/README.md)**

---

</div>

## 📖 Executive Summary

**Noxfort Monitor™** is an industrial event-driven observability platform developed in Go to monitor distributed systems, autonomous agents (such as **Synapse** and **Carina**), and industrial IoT hardware.

It features a dual-engine database persistence layer (**PostgreSQL** for enterprise network operations and embedded **SQLite** in WAL mode with live hot-reloading), **Zero-Trust** telemetry ingestion via **MQTT** and **HTTP REST** (`POST /api/pair` + `X-Device-Token`), silent failure detection through the **Watchdog Engine**, multi-channel alert dispatching with retry backoff and strict Role-Based Access Control (**RBAC**), persistent SHA-256 hashed sessions with brute-force lockout, **DuckDNS + Caddy** automatic HTTPS / **Ngrok** reverse tunneling, automated data retention & hot backups/restore, and a native desktop interface powered by **Wails v2** with **Headless** and **Demo** server modes.

---

## 📚 Documentation Hub & Knowledge Vault

Explore the full architecture, internal mechanics, and developer guides for the Noxfort Monitor ecosystem:

| Card / Subsystem | Focus Area | Direct Link |
| :--- | :--- | :---: |
| 📚 **Documentation Hub** | Central Multi-Language Index & Direct Topic Navigation Matrix | [Explore Hub](docs/README.md) |
| 🗺️ **Master MOC** | Obsidian Knowledge Graph, Directory Map & Technical Architecture | [View MOC](docs/INDEX.md) |
| 🏛️ **Core Architecture** | Event-Driven Architecture (EDA), SOLID Layers & Goroutine Concurrency | [View Blueprint](ARCHITECTURE.md) |
| 📡 **API & Protocols** | Zero-Trust Pairing (`POST /api/pair`), MQTT (`noxfort/telemetry/#`) & HTTP REST (`:22100`) | [View API Reference](docs/API_REFERENCE.md) |
| 🗄️ **Dual-Engine Persistence** | PostgreSQL & SQLite WAL, `DBManager` Hot-Reload, Backup/Restore & Retention | [View DB Specs](docs/DATABASE.md) |
| 🔐 **Security & RBAC** | Read-Only Operator RBAC, Persistent Sessions, Brute-Force Lockout & PBKDF2 | [View Security](docs/SECURITY.md) |
| 🌐 **Remote Access & WAN** | DuckDNS Dynamic DNS & Embedded Ngrok Tunnel for Edge Ingestion | [View Remote Access](docs/REMOTE_ACCESS.md) |
| 🛡️ **Caddy & Automatic HTTPS** | Reverse Proxy with DuckDNS DNS-01 ACME Challenge & WSS MQTT Proxy | [View Caddy Guide](docs/CADDY_INTEGRATION.md) |
| 🖥️ **Desktop App & Headless** | Wails v2 Native GUI, Kernel Flock Lock, Systray & Headless Telemetry Daemon | [View Desktop Guide](docs/DESKTOP_APP.md) |
| 🔍 **Audit Trail & SLA** | Triple Audit: Security Logs, Alert Dispatch SLA & Equipment Downtime | [View Audit Trail](docs/AUDIT_TRAIL.md) |
| 🚀 **Production Deployment** | Systemd Service, NGINX / Caddy Reverse Proxy & Debian `.deb` Package | [View Deployment](docs/DEPLOYMENT.md) |
| 🧪 **Testing & Diagnostics** | Unit Tests with Repository Mocks, `mosquitto_pub` & `curl` Verification | [View Testing](docs/TESTING.md) |
| 👨‍💻 **Developer Guides** | Local Environment, Dependency Injection & Extensibility | [View Dev Guides](docs/DEVELOPER_GUIDES.md) |
| 🔬 **Research Notes** | Architectural Decisions, CGO Elimination & Clustering Roadmap | [View Decisions](docs/RESEARCH_NOTES.md) |

---

## ⚡ Core Architecture Highlights

- **Zero-Trust Telemetry Ingestion (MQTT + HTTP REST):** Non-blocking concurrent processing over MQTT `tcp://127.0.0.1:1883` (`noxfort/telemetry/#`, enforced by `MONITOR_MQTT_STRICT`) and HTTP REST `POST /api/telemetry` on port `22100` with out-of-band device pairing (`POST /api/pair`, 5-minute TTL, `nox_live_` API keys via `X-Device-Token` / `Bearer`) and ±5m anti-replay validation. Direct web browser access to HTML routes on the external port is blocked (`403 Forbidden`).
- **Live Dashboard & Intelligent Noise Filter:** Real-time dashboard polling (`GET /api/dashboard/summary` every 10s) classifying devices across 5 lifecycle states (`ONLINE`, `OFFLINE`, `PENDING`, `PAUSED`, `WAITING`), while keep-alive signals ("*system ok*", "*heartbeat*", "*online*") update heartbeats without alert noise and Maintenance Mode (`Enabled=false`) suppresses paused devices.
- **Dual-Engine Persistence (PostgreSQL & SQLite WAL):** Central `DBManager` orchestrates runtime database switching with single-transaction `MigrateData` across SQLite (WAL journaling + busy timeout) and PostgreSQL without dropping connections or restarting.
- **High-Throughput Telemetry Buffering:** Dedicated `BufferedTelemetryWriter` worker handles asynchronous batch flushes to storage, exporting live `noxfort_telemetry_buffer_depth` metrics.
- **CGO-Free Embedded Engine:** Built using `modernc.org/sqlite`, completely removing CGO compiler and C-toolchain dependencies during cross-compilation.
- **Watchdog Engine & Reliable Alert Dispatch:** Dedicated 30-second ticker evaluating `LastSeen` heartbeats (> 5m outage detection + expired `PENDING` pairing cleanup); concurrent Email (SMTP with implicit TLS on port 465 and CRLF sanitization) and Telegram (MarkdownV2) dispatching with retry backoff, graceful shutdown draining, and `telemetry_id` audit linkage.
- **Strict Operator RBAC, Persistent Sessions & Brute-Force Lockout:** `OPERATOR` accounts operate in strictly read-only mode (blocked from mutations and DB backups with content-negotiated JSON or `forbidden.html` 403 views); `DBSessionStore` persists SHA-256 session token hashes in DB; `LoginAttemptTracker` enforces account+IP brute-force lockouts (`429 Too Many Requests`); HTML templates redact stored secrets (`SMTPPass`, `TelegramBotToken`, `DuckDNSToken`, active `ApiKey`).
- **Remote Access & Automatic HTTPS (DuckDNS & Caddy / Ngrok):** Native dynamic DNS updates via DuckDNS, automated Let's Encrypt TLS certificates using Caddy with DNS-01 ACME validation and `X-Forwarded-*` trust (`TRUSTED_PROXIES`), plus optional Ngrok reverse tunneling.
- **Native Observability & Prometheus Metrics:** Dedicated `/healthz` (liveness), `/api/health` (component diagnostics & DB latency), and `/metrics` (Prometheus exposition including `noxfort_devices_by_status`, `noxfort_telemetry_ingested_total`, `noxfort_telemetry_rejected_total`, `noxfort_alerts_dispatched_total`, and `noxfort_telemetry_buffer_depth`).
- **Scheduled Hot Backups, One-Click Restore & Retention Pruning:** Atomic snapshotting via SQLite `VACUUM INTO` and `pg_dump` with scheduled execution (`MONITOR_BACKUP_INTERVAL_HOURS`), secure download/restore (`PRAGMA quick_check` / `pg_restore`) via Web UI, and automated data pruning (`RetentionManager`, `MONITOR_RETENTION_DAYS=90`).
- **Hardened MQTT & PBKDF2 Credentials:** Enforced Mosquitto authentication (`allow_anonymous false`) via `make broker-auth`, PBKDF2 HMAC-SHA256 user password hashing (100,000 iterations), and background MQTT reconnect on boot failures or `/settings` changes.
- **Unified CI/CD & Frontend Quality:** GitHub Actions workflow executing `go vet`, race-detected Go tests, secret leakage audits, and sub-second Vitest frontend DOM tests.
- **Wails v2 Desktop GUI, Headless & Demo Modes:** WebKitGTK desktop interface with atomic kernel file locking (`syscall.Flock`) and minimize-to-tray, `--headless` server daemon mode, or `--demo` HTTP UI mode for automated Playwright recordings (`make record-demo`).

---

## 🚀 Quick Start

### 1. Prerequisites
Ensure you have [Go 1.22+](https://go.dev/dl/), Node.js 18+, and WebKitGTK libraries installed on Ubuntu/Debian:
```bash
sudo apt-get update && sudo apt-get install -y \
  libgtk-3-dev libwebkit2gtk-4.1-dev libappindicator3-dev mosquitto nodejs npm
```

### 2. Configure & Start Authenticated MQTT Broker
```bash
# Generate secure PBKDF2 credentials for Mosquitto & sync into .env
make broker-auth

# Start the Mosquitto daemon
make broker-start
```

### 3. Run Automated Tests & Static Analysis
```bash
# Run both Backend Go unit tests (-race) and Frontend Vitest suites
make test

# Run Go static analysis (go vet)
make lint
```

### 4. Run in Development Mode (Desktop GUI)
```bash
make build
make run
```

### 5. Run in Server Mode (Headless Daemon)
Ideal for cloud instances and screenless server environments:
```bash
make run-headless
# Or run the compiled binary directly:
./bin/noxfort-monitor --headless
```

### 6. Create Database Backups
```bash
# Generate a hot, consistent database snapshot
make backup

# List all current backups
make backup-list
```

### 7. Generate Debian Installer (`.deb`)
```bash
make deb
sudo dpkg -i build_deb/noxfort-monitor_2.0.1_amd64.deb
```

---

## 🔐 Authentication & Default Credentials

* **Operator Interface**: Accessible exclusively via the native desktop application window (or `--demo` flag for automated demo recording). Direct web browser access over external HTTP ports is disabled with HTTP 403 Forbidden; external port `22100` serves `POST /api/pair`, `POST /api/telemetry`, `/healthz`, `/api/health`, and `/metrics`.
* **First-Access Default Credentials**: If no `.env` file is present, the default bootstrap credentials for logging into the desktop application are:
  * **Username**: `admin`
  * **Password**: `Admin@Noxfort2026!`
* **Production Configuration**: Copy `.env.example` to `.env` and set strong administrator credentials before deployment:
  ```bash
  cp .env.example .env
  chmod 600 .env
  # Set MONITOR_ADMIN_USER, MONITOR_ADMIN_PASSWORD, MQTT_USER, MQTT_PASSWORD
  # Optional tuning: MONITOR_MQTT_STRICT, MONITOR_RETENTION_DAYS, MONITOR_BACKUP_INTERVAL_HOURS, TRUSTED_PROXIES
  ```
* **Persistence Location**: In local SQLite mode, the database resides at `~/Documentos/Monitor/monitor_logs.db` (backups under `~/Documentos/Monitor/backups/`). In PostgreSQL mode, data resides on the configured database server.

---

<div align="center">
  <img src="docs/assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="48" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Industrial Telemetry Ingestion, Observability & Incident Response Orchestration • Noxfort Monitor™ Server v2.0</i><br/>
  <small>Licensed under the <a href="LICENSE">GNU Affero General Public License v3.0 or later</a> (SPDX: <code>AGPL-3.0-or-later</code>). © 2026 Noxfort Systems.</small>
</div>
