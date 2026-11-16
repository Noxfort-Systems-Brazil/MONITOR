---
tags: [home, index, monitor, hub]
aliases: [Documentation Index, Overview, Portal]
---

# 🛡️ Noxfort Monitor™ Documentation Hub

Welcome to the technical documentation library for **Noxfort Monitor™ v2.0** — an industrial event-driven telemetry ingestion, observability, and incident response platform developed in Go.

Designed to operate seamlessly on **GitHub**, **MkDocs Material**, and as an **[Obsidian](https://obsidian.md/) Knowledge Vault**, this documentation suite covers the event-driven Go concurrency model, Zero-Trust device pairing (`POST /api/pair` + `X-Device-Token`), dual-engine PostgreSQL & SQLite WAL persistence with automated retention & hot backup/restore, multi-channel alert dispatching (Telegram & SMTP TLS 465) with retry backoff, strict read-only Operator RBAC with persistent SHA-256 sessions and brute-force lockout, **DuckDNS + Caddy** automatic HTTPS / **Ngrok** reverse tunneling, Prometheus observability, and native desktop integration powered by **Wails v2**.

---

## 🗺️ Master Navigation & Modules

| Subsystem / Dimension | Focus Area | Direct Link |
| :--- | :--- | :---: |
| 📚 **Master Map of Content** | Primary Obsidian Hub & Codebase Directory Map | [Explore Hub](INDEX.md) |
| 🌐 **Multi-Language Hub** | Translations across English, Portuguese, Spanish, French, Russian & Chinese | [Language Hub](README.md) |
| 🏛️ **System Architecture** | Event-Driven Architecture, Dependency Injection & Goroutine Concurrency | [View Blueprint](../ARCHITECTURE.md) |
| 📡 **API & Ingestion Protocols** | MQTT (`noxfort/telemetry/#`), Zero-Trust Pairing (`/api/pair`), HTTP REST & Prometheus `/metrics` | [View API Reference](API_REFERENCE.md) |
| 🗄️ **Dual-Engine Persistence** | PostgreSQL & SQLite WAL, `DBManager` Hot-Reload, Backup/Restore & Retention | [View DB Specs](DATABASE.md) |
| 🔐 **Security & RBAC** | Read-Only Operator RBAC, Persistent Sessions, Brute-Force Lockout & PBKDF2 | [View Security](SECURITY.md) |
| 🌐 **Remote Access & Tunnel** | DuckDNS Dynamic DNS, Ngrok Reverse Tunnel & WAN Ingestion | [View Remote Access](REMOTE_ACCESS.md) |
| 🛡️ **Caddy & DuckDNS (HTTPS)** | Automatic TLS/SSL with DNS-01 ACME, `X-Forwarded-*` & WSS MQTT Proxy | [View Caddy Guide](CADDY_INTEGRATION.md) |
| 🖥️ **Desktop App & Headless** | Wails v2, WebKitGTK, Unix IPC Lock, Systray, `--headless` & `--demo` Modes | [View Desktop Guide](DESKTOP_APP.md) |
| 🔍 **Audit Trail & SLA** | Triple Audit: Security Logs, Alert Dispatch SLA & Equipment Downtime | [View Audit Trail](AUDIT_TRAIL.md) |
| 🚀 **Production Deployment** | Systemd Headless Daemon, Caddy/NGINX Proxy, Env Tuning & `.deb` Package | [View Deployment](DEPLOYMENT.md) |
| 🧪 **Testing & Diagnostics** | Go Backend Unit Tests, Vitest Frontend DOM Suite, `mosquitto_pub` & `curl` | [View Testing](TESTING.md) |
| 👨‍💻 **Developer Guides** | Local Environment, Dependency Injection & Extensibility | [View Dev Guides](DEVELOPER_GUIDES.md) |
| 🔬 **Technical Decisions** | Architectural Decisions, Zero-Trust Onboarding, Sessions & Instrumentation | [View Decisions](RESEARCH_NOTES.md) |

---

## ⚡ Quick Architecture Summary

```text
       ┌────────────────────────────────────────────────────────┐
       │             Noxfort Monitor Go Process                 │
       │              (cmd/server/main.go)                      │
       └──────────────────────────┬─────────────────────────────┘
                                  │
      ┌───────────────────────────┼───────────────────────────┐
      ▼                           ▼                           ▼
┌──────────────┐          ┌──────────────┐          ┌──────────────────┐
│ MQTT Broker  │          │ HTTP Server  │          │ DuckDNS + Caddy  │
│(noxfort/     │ ◄──────► │(:22100 REST/ │ ◄──────► │ / Ngrok Tunnel   │
│ telemetry/#) │          │ Pair/Metrics)│          │ (WAN HTTPS/WSS)  │
└──────┬───────┘          └──────┬───────┘          └──────────────────┘
       │ (Zero-Trust Policy)     │ (X-Device-Token + Anti-Replay)
       ▼                         ▼
┌──────────────┐          ┌──────────────┐          ┌──────────────────┐
│ StateManager │ ◄──────► │   Watchdog   │ ◄──────► │  AlertService    │
│(Maint/Filter)│          │(30s / TTL GC)│          │(Retry + TLS 465) │
└──────┬───────┘          └──────┬───────┘          └────────┬─────────┘
       │                         │                           │
       └─────────────────────────┼───────────────────────────┘
                                 ▼
                    ┌─────────────────────────┐
                    │  Central DBManager      │
                    │ (PG <-> SQLite WAL)     │
                    │ Sessions/Backup/Prune   │
                    └─────────────────────────┘
```

---

<div align="center">
  <img src="assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Industrial Telemetry & Observability • Noxfort Monitor™ Server v2.0</i><br/>
  <small>Licensed under the <a href="../LICENSE">GNU Affero General Public License v3.0 or later</a> (SPDX: <code>AGPL-3.0-or-later</code>). © 2026 Noxfort Systems.</small>
</div>
