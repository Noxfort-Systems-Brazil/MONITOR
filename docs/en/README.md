# Noxfort Monitor — Documentation Hub (English)

Welcome to the English documentation suite for **Noxfort Monitor**.

## Guides

| Guide | Summary |
| :--- | :--- |
| [System Architecture](./architecture.md) | Modular Go architecture, execution modes (`desktop`, `--headless`, `--demo`), MQTT/HTTP pipelines, and graceful shutdown. |
| [API & Protocol Reference](./api_reference.md) | `noxfort/telemetry/#` MQTT spec, Zero-Trust `/api/pair` & `/api/telemetry`, RBAC endpoints, and Prometheus `/metrics`. |
| [Security Architecture](./security.md) | Strictly read-only `OPERATOR`, `PBKDF2` hashing, `DBSessionStore`, `LoginAttemptTracker` lockout, proxy trust, and secret redaction. |
| [Database & Lifecycle](./database.md) | SQLite WAL + PostgreSQL hot switching, `7` `ReloadableRepository` implementations, transactional `MigrateData`, backup/restore, and `RetentionManager`. |
| [Audit Trail & Compliance](./audit_trail.md) | `SecurityAuditLog`, `AlertDispatchLog` (`telemetry_id`), `DeviceStateTransition`, `Operator+` access, and 90-day pruning. |
| [Deployment & Operations](./deployment.md) | `.env` configuration (`configs/config.yaml` removed), Docker Compose, NGINX headers, scheduled backups, and metrics scraping. |
| [Caddy Reverse Proxy](./caddy_integration.md) | `caddy/Caddyfile`, `TRUSTED_PROXIES`, `MONITOR_USE_HTTPS`, and testing Zero-Trust HTTPS telemetry. |
| [Remote Access & Tunnels](./remote_access.md) | DuckDNS & Ngrok setup via `/remote`, HTML credential redaction, and secure edge onboarding. |
| [Desktop App & Modes](./desktop_app.md) | Wails GUI vs. `--demo` vs. `--headless`, `forbidden.html`, 10s `/api/dashboard/summary` polling, and self-service password updates. |
| [Testing & Verification](./testing.md) | `make test` (Go + Vitest), `go vet`, `mosquitto_pub`, Zero-Trust `curl` checks, and `make record-demo`. |
| [Developer Guides](./developer_guides.md) | `Makefile` workflow, `ReloadableRepository`, HTTP security helpers, `internal/metrics`, and shutdown context propagation. |
| [Architectural Decisions (ADRs)](./research_notes.md) | Design rationale behind `.env`, Zero-Trust pairing, hashed DB sessions, WAL/retention, and alert traceability. |
