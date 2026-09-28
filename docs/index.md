---
tags: [home, index, monitor, hub]
aliases: [Documentation Index, Overview, Portal]
---

# 🛡️ Noxfort Monitor™ Documentation Hub

Welcome to the technical documentation library for **Noxfort Monitor™ v2.0** — an industrial event-driven telemetry ingestion, observability, and incident response platform developed in Go.

Designed to operate seamlessly on **GitHub** and as an **[Obsidian](https://obsidian.md/) Knowledge Vault**, this documentation suite covers the event-driven Go concurrency model, dual-engine PostgreSQL & SQLite persistence, multi-channel alert dispatching (Telegram & SMTP), secure reverse tunneling via Ngrok, and native desktop integration powered by Wails v2.

---

## 🗺️ Master Navigation & Modules

| Subsystem / Dimension | Focus Area | Direct Link |
| :--- | :--- | :---: |
| 📚 **Master Map of Content** | Primary Obsidian Hub & Codebase Directory Map | [Explore Hub](INDEX.md) |
| 🌐 **Multi-Language Hub** | Translations across English, Portuguese, Spanish, French, Russian & Chinese | [Language Hub](README.md) |
| 🏛️ **System Architecture** | Event-Driven Architecture, Dependency Injection & Goroutine Concurrency | [View Blueprint](../ARCHITECTURE.md) |
| 📡 **API & Ingestion Protocols** | MQTT Ingestion (`:1883`) & HTTP REST Ingestion (`/api/telemetry` at `:22100`) | [View API Reference](API_REFERENCE.md) |
| 🗄️ **Dual-Engine Persistence** | PostgreSQL & SQLite, Live Hot-Reload (`DBManager`) & Zero-Downtime Migration | [View DB Specs](DATABASE.md) |
| 🔐 **Security & RBAC** | Session Cookies, Salted Password Hashing, Superuser Bootstrap & RBAC | [View Security](SECURITY.md) |
| 🌐 **Remote Access & Tunnel** | Ngrok Reverse Tunnel, Firewall/CGNAT Traversal & WAN Ingestion | [View Remote Access](REMOTE_ACCESS.md) |
| 🖥️ **Desktop App & Headless** | Wails v2, WebKitGTK, Unix IPC Socket Lock, Systray & Headless Server Daemon | [View Desktop Guide](DESKTOP_APP.md) |
| 🔍 **Audit Trail & SLA** | Triple Audit: Security Logs, Alert Dispatch SLA & Equipment Downtime | [View Audit Trail](AUDIT_TRAIL.md) |
| 🚀 **Production Deployment** | Systemd Headless Daemon, NGINX SSL Reverse Proxy & Debian `.deb` Package | [View Deployment](DEPLOYMENT.md) |
| 🧪 **Testing & Diagnostics** | Automated Pytest/Go Unit Tests, Repository Mocks, `mosquitto_pub` & `curl` | [View Testing](TESTING.md) |
| 👨‍💻 **Developer Guides** | Local Environment, Dependency Injection & Extensibility | [View Dev Guides](DEVELOPER_GUIDES.md) |
| 🔬 **Technical Decisions** | Architectural Decisions, CGO Elimination & Clustering Roadmap | [View Decisions](RESEARCH_NOTES.md) |

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
│ MQTT Broker  │          │ HTTP Server  │          │   Ngrok Tunnel   │
│(tcp :1883)   │ ◄──────► │(REST :22100) │ ◄──────► │ (WAN Ingestion)  │
└──────┬───────┘          └──────┬───────┘          └──────────────────┘
       │                         │
       ▼                         ▼
┌──────────────┐          ┌──────────────┐          ┌──────────────────┐
│ StateManager │ ◄──────► │   Watchdog   │ ◄──────► │  AlertService    │
│(Filter & Act)│          │ (Silent Fail)│          │(Email / Telegram)│
└──────┬───────┘          └──────┬───────┘          └────────┬─────────┘
       │                         │                           │
       └─────────────────────────┼───────────────────────────┘
                                 ▼
                    ┌─────────────────────────┐
                    │  Central DBManager      │
                    │ (PostgreSQL <-> SQLite) │
                    └─────────────────────────┘
```
