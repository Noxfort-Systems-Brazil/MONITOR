<div align="center">

<img src="../assets/monitor-logo.png" alt="Noxfort Monitor Logo" width="120" />

# Noxfort Monitor™ — Technical Documentation Suite
### System Architecture, Ingestion Protocols & Dual-Engine Persistence
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Active-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![License: AGPL-3.0-or-later](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](../../LICENSE)

---

🌐 **Translations / Languages:** **[🇺🇸 English](README.md)** • **[🇧🇷 Português](../pt-br/README.md)** • **[🇪🇸 Español](../es/README.md)** • **[🇫🇷 Français](../fr/README.md)** • **[🇷🇺 Русский](../ru/README.md)** • **[🇨🇳 简体中文](../zh/README.md)** • **[📚 Central Hub](../README.md)**

---

</div>

## Welcome to the Official Technical Documentation

This directory contains the canonical **English** technical documentation suite for **Noxfort Monitor™ v2.0** — an industrial event-driven observability, telemetry ingestion, and incident orchestration platform developed in Go for distributed systems, autonomous agents (**Carina**, **Synapse**), and IoT field nodes.

## Technical Guides Directory

| Document | Subject & Scope | Key Topics |
|---|---|---|
| 📖 **[System Architecture](architecture.md)** | Core Architecture Blueprint | Event-Driven Architecture (EDA), goroutine concurrency, SOLID layer decoupling, and composition root dependency injection. |
| 📡 **[API & Ingestion Protocols](api_reference.md)** | Ingestion Specifications | Non-blocking MQTT ingestion (`:1883`), HTTP REST `POST /api/telemetry` (`:22100`), universal `IncomingEvent` JSON schema, and management APIs. |
| 🗄️ **[Dual-Engine Persistence](database.md)** | Database Subsystem | Live connection hot-reload via `DBManager`, PostgreSQL & CGO-free pure-Go SQLite, automatic data migrator, and dialect query adapters. |
| 🔐 **[Security & RBAC](security.md)** | Authentication & Security | Role-Based Access Control (`RoleAdmin`, `RoleOperator`), salted password hashing, thread-safe session cookies, and idempotent superuser bootstrapping. |
| 🌐 **[Remote Access (DuckDNS & Ngrok)](remote_access.md)** | WAN Reverse Tunnel | Dynamic DNS synchronization via DuckDNS, reverse tunneling via Ngrok across industrial firewalls & CGNAT, and remote telemetry forwarding. |
| 🛡️ **[Caddy Reverse Proxy & DuckDNS](caddy_integration.md)** | Automated HTTPS Gateway | Enterprise TLS/SSL termination with DNS-01 ACME challenge, port 443 standardization, and encrypted WebSockets (`/mqtt`). |
| 🖥️ **[Desktop App & Headless](desktop_app.md)** | UI & Operations | Wails v2 + native WebKitGTK runtime, Unix IPC single-instance socket lock, system tray lifecycle, and `--headless` server daemon mode. |
| 🔍 **[Audit Trail & SLA](audit_trail.md)** | Compliance & Observability | Triple audit model: security access logs (`SecurityAuditLog`), alert delivery SLA tracking (`AlertDispatchLog`), and downtime monitoring. |
| 🚀 **[Production Deployment](deployment.md)** | DevOps & Production | Standalone systemd headless daemon, NGINX/Caddy reverse proxy configuration, and native Debian `.deb` installer packaging. |
| 🧪 **[Testing & QA](testing.md)** | Quality Assurance | Automated Go unit tests with repository mocks, manual verification with `mosquitto_pub` / `curl`, and notification diagnostics. |
| 👨‍💻 **[Advanced Developer Guide](developer_guides.md)** | Engineering & Extensibility | Local setup, SOLID conventions, reloadable repository patterns (`DBManager`), custom notification channels, and entities. |
| 🔬 **[Research Notes & Technical Decisions](research_notes.md)** | Architectural Records (ADRs) | Architecture decision log: pure-Go SQLite migration, Wails v2, DuckDNS/Caddy reverse proxy, and gRPC clustering roadmap. |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Industrial Telemetry & Observability • Noxfort Monitor™ Server v2.0</i><br/>
  <small>Licensed under the <a href="../../LICENSE">GNU Affero General Public License v3.0 or later</a> (SPDX: <code>AGPL-3.0-or-later</code>). © 2026 Noxfort Systems.</small>
</div>
