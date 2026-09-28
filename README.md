<div align="center">

<img src="docs/assets/monitor-logo.png" alt="Noxfort Monitor Logo" width="130" />

# NOXFORT MONITOR™
### Industrial Telemetry Ingestion, Observability & Incident Response Orchestration
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Active-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![License](https://img.shields.io/badge/License-AGPL_v3-blue?style=flat)](LICENSE)

[![Noxfort Monitor GitHub Repository Card](https://github-readme-stats.vercel.app/api/pin/?username=Noxfort-Systems-Brazil&repo=MONITOR&theme=dark)](https://github.com/Noxfort-Systems-Brazil/MONITOR)

---

🌐 **Translations / Idiomas:** **[🇺🇸 English](README.md)** • **[🇧🇷 Português do Brasil](docs/pt-br/README.md)** • **[🇪🇸 Español](docs/es/README.md)** • **[🇫🇷 Français](docs/fr/README.md)** • **[🇷🇺 Русский](docs/ru/README.md)** • **[🇨🇳 简体中文](docs/zh/README.md)** • **[📚 Documentation Hub](docs/README.md)**

---

</div>

## 📖 Executive Summary

**Noxfort Monitor™** is an industrial event-driven observability platform developed in Go to monitor distributed systems, autonomous agents (such as **Synapse** and **Carina**), and industrial IoT hardware.

It features a dual-engine database persistence layer (**PostgreSQL** for enterprise network operations and embedded **SQLite** with live hot-reloading), concurrent telemetry ingestion via **MQTT** and **HTTP REST**, silent failure detection through the **Watchdog Engine**, multi-channel alert dispatching with Role-Based Access Control (**RBAC**), secure reverse tunneling via **Ngrok** for edge nodes on remote networks, and a native desktop interface powered by **Wails v2** with **Headless** server mode support.

---

## 📚 Documentation Hub & Knowledge Vault

Explore the full architecture, internal mechanics, and developer guides for the Noxfort Monitor ecosystem:

| Card / Subsystem | Focus Area | Direct Link |
| :--- | :--- | :---: |
| 📚 **Documentation Hub** | Central Multi-Language Index & Direct Topic Navigation Matrix | [Explore Hub](docs/README.md) |
| 🗺️ **Master MOC** | Obsidian Knowledge Graph, Directory Map & Technical Architecture | [View MOC](docs/INDEX.md) |
| 🏛️ **Core Architecture** | Event-Driven Architecture (EDA), SOLID Layers & Goroutine Concurrency | [View Blueprint](ARCHITECTURE.md) |
| 📡 **API & Protocols** | High-throughput MQTT (`tcp://:1883`) & HTTP REST Ingestion (`:22100`) | [View API Reference](docs/API_REFERENCE.md) |
| 🗄️ **Dual-Engine Persistence** | PostgreSQL & SQLite, Live Hot-Reload (`DBManager`) & Zero-Downtime Migration | [View DB Specs](docs/DATABASE.md) |
| 🔐 **Security & RBAC** | RoleAdmin vs RoleOperator, Salted Hashes, Session Cookies & Bootstrapping | [View Security](docs/SECURITY.md) |
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

- **Dual Telemetry Ingestion (MQTT + HTTP REST):** Non-blocking concurrent processing over MQTT `tcp://127.0.0.1:1883` and HTTP REST `POST /api/telemetry` on port `22100`. Direct web browser access to HTML routes is blocked (403 Forbidden) to guarantee control operations remain confined to the native desktop application.
- **Intelligent Noise Filter:** Incoming keep-alive signals ("*system ok*", "*heartbeat*", "*online*") update device timestamps in memory without creating unneeded database writes or alert noise.
- **Dual-Engine Persistence (PostgreSQL & SQLite):** Central `DBManager` orchestrates runtime database switching via the control console without dropping active client connections or restarting the process.
- **High-Throughput Telemetry Buffering:** Dedicated `BufferedTelemetryWriter` worker handles asynchronous batch flushes to storage, preventing I/O bottlenecks under heavy IoT ingest loads.
- **CGO-Free Embedded Engine:** Built using `modernc.org/sqlite`, completely removing CGO compiler and C-toolchain dependencies during cross-compilation.
- **Watchdog Engine (Silent Failure Detection):** Dedicated ticker evaluating `LastSeen` heartbeats; synthesizes `CRITICAL` `System OFFLINE` incidents when nodes go silent for > 5 minutes and auto-resolves when transmission resumes.
- **Role-Based Alert Routing (RBAC):** Goroutine-based concurrent alert dispatching via Email (SMTP) and Telegram Bot (MarkdownV2) segregated by responsibility (Hardware $\rightarrow$ Technicians, Software $\rightarrow$ Programmers, Global $\rightarrow$ Administrators).
- **Remote Access & Automatic HTTPS (DuckDNS & Caddy / Ngrok):** Native dynamic DNS updates via DuckDNS, automated Let's Encrypt TLS certificates using Caddy with DNS-01 ACME validation, plus optional Ngrok reverse tunneling for edge nodes behind firewalls and CGNAT.
- **Native Observability & Prometheus Metrics:** Dedicated `/healthz` (liveness), `/api/health` (component diagnostics & DB latency), and `/metrics` (Prometheus exposition format) endpoints for real-time operational monitoring.
- **Zero-Downtime Hot Backups & Retention:** Atomic snapshotting via SQLite `VACUUM INTO` and PostgreSQL support with automated 7-day retention rotation, accessible via Web UI or `make backup`.
- **Hardened MQTT & PBKDF2 Credentials:** Enforced authentication (`allow_anonymous false`) with automated password generation (`make broker-auth`) and Paho client credential integration.
- **Unified CI/CD & Frontend Quality:** GitHub Actions workflow executing `go vet`, race-detected Go tests, secret leakage audits, and sub-second Vitest frontend DOM tests.
- **Wails v2 Desktop GUI & Headless Mode:** WebKitGTK desktop interface with atomic kernel file locking (`syscall.Flock`) single-instance enforcement and minimize-to-tray; or screenless background ingestion daemon using `--headless`.

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

* **Operator Interface**: Accessible exclusively via the native desktop application window. (Direct web browser access over external HTTP ports is disabled with HTTP 403 Forbidden; external port `22100` serves `POST /api/telemetry` for IoT nodes).
* **First-Access Default Credentials**: If no `.env` file is present, the default bootstrap credentials for logging into the desktop application are:
  * **Username**: `admin`
  * **Password**: `admin`
* **Production Configuration**: Copy `.env.example` to `.env` and set strong administrator credentials before deployment:
  ```bash
  cp .env.example .env
  chmod 600 .env
  # Set MONITOR_ADMIN_USER, MONITOR_ADMIN_PASSWORD, MQTT_USER, MQTT_PASSWORD
  ```
* **Persistence Location**: In local SQLite mode, the database resides at `~/Documentos/Monitor/monitor_logs.db`. In PostgreSQL mode, data resides on the configured database server.

---

<div align="center">
  <img src="docs/assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="48" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Industrial Telemetry Ingestion, Observability & Incident Response Orchestration • Noxfort Monitor™ Server v2.0</i><br/>
  <small>Licensed under the <a href="LICENSE">GNU Affero General Public License v3.0</a>. © 2026 Noxfort Systems.</small>
</div>
