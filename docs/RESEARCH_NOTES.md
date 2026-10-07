[📚 Documentation Hub](INDEX.md) > **Research Notes & Technical Decisions**

---

# 🔬 Research Notes & Technical Decisions: Noxfort Monitor™

This document serves as an architectural decision record (ADR), benchmark repository, and research backlog for the future evolution of **Noxfort Monitor™**.

---

## 1. Persistence: From CGO Dependency to Dual-Engine Architecture

### Context
Earlier versions of Monitor relied exclusively on `mattn/go-sqlite3`, requiring a C compiler (CGO) to build the binary. This introduced substantial friction for cross-compilation, CI/CD pipelines, and package distribution.

### Decision
1. **Migration to Pure-Go SQLite**: Adopted `modernc.org/sqlite` to enable 100% pure Go compilation in embedded deployments.
2. **Evolution to Dual-Engine (SQLite + PostgreSQL)**: In enterprise industrial facilities, multiple operators access the dashboard simultaneously, and high-concurrency writes to SQLite created lock contention (`database is locked`).
3. **Zero-Downtime Hot-Reload**: Implemented [`DBManager`](DATABASE.md) to enable dynamic switching between SQLite and PostgreSQL at any time via the graphical UI, with automated record migration and without disrupting telemetry ingestion.

---

## 2. Desktop Interface: From Simple Systray to Wails v2

### Context
The initial implementation used a minimal system tray icon (`getlantern/systray`) that launched the operating system's default browser. Users reported session conflicts with other web applications, lack of dedicated shortcuts, and the absence of a cohesive desktop experience.

### Decision
Migrated to **Wails v2** ([`docs/DESKTOP_APP.md`](DESKTOP_APP.md)):
* **Native WebKitGTK**: Fast, consistent rendering of HTML5/Bootstrap without the memory overhead of Chromium-based browsers.
* **Integrated Lifecycle**: The system tray was rewritten to live within the Wails desktop event loop, supporting minimize-on-close (`HideWindowOnClose`) and one-click restoration.
* **Session Isolation**: A header interceptor synchronizes the `noxfort_session` cookie directly in desktop memory.
* **Headless Compatibility**: Added `--headless` mode to allow the exact same binary to run as a headless Linux server daemon without display initialization failures.

---

## 3. WAN Connectivity: DuckDNS + Caddy Edge Gateway vs Ngrok Tunnel

### Context
Industrial edge agents (**Synapse**, **Carina**) and remote IoT nodes operate across WAN boundaries outside the server's local subnet. Initially, Monitor relied on **Ngrok** outbound tunnels to bypass CGNAT and strict router firewalls. However, free-tier tunnel services introduce bandwidth constraints, session timeouts, and dependencies on third-party cloud intermediaries.

### Decision
Evolved the [`internal/tunnel`](REMOTE_ACCESS.md) and [`caddy/`](CADDY_INTEGRATION.md) subsystems into a dual-connectivity architecture:
1. **DuckDNS Dynamic DNS + Containerized Caddy (Primary / Recommended)**:
   * Native Go updater dynamically synchronizes public IPv4 and IPv6 addresses with DuckDNS.
   * Containerized Caddy acts as an automated edge gateway executing **ACME DNS-01 challenges** against the DuckDNS API.
   * Eliminates the requirement to open port 80 for TLS validation while terminating encrypted traffic on standard port 443.
   * Proxies telemetry to `POST /api/telemetry` on port 22100 and exposes encrypted WebSockets (`wss://domain.duckdns.org/mqtt`) for Mosquitto.
2. **Ngrok Reverse Tunnel (Fallback / Zero-Forwarding)**:
   * Retained as a secondary, zero-configuration driver for installations behind restrictive carrier-grade NAT (CGNAT) where inbound ports cannot be opened.

---

## 4. Roadmap & Future Research

### 4.1 gRPC Expansion for Monitor Federation
While MQTT and HTTP REST comfortably serve sensor telemetry, multi-server federation (active-passive clustering or HQ-to-branch hierarchies) will benefit from **gRPC / Protocol Buffers**:
* Drastic reduction of network payload overhead over constrained satellite links.
* Strictly typed, immutable schemas for audit log replication.

### 4.2 Predictive Anomaly Detection
Current alerts trigger via static rules and thresholds (e.g., `temperature > 80°C`). We are investigating lightweight time-series forecasting in Go (such as exponential moving averages and embedded ONNX runtimes) to identify failure trends before operational safety limits are breached.

---

### 🔗 Related Documentation
* 🏗️ [System Architecture](../ARCHITECTURE.md) — Macro-level system design
* 🗄️ [Database & Persistence](DATABASE.md) — Dual-engine implementation
* 🖥️ [Desktop Application](DESKTOP_APP.md) — Wails architecture details
* 🌐 [Remote Access](REMOTE_ACCESS.md) — Reverse tunnel rationale
