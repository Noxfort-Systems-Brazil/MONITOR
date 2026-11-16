# Noxfort Monitor — Architectural Decision Records (ADRs)

This document captures core architectural decisions, security trade-offs, and evolution notes for Noxfort Monitor.

---

## ADR-01: Removal of Static `configs/config.yaml` in Favor of `.env` + SQL `settings`

* **Context**: Maintaining duplicate configuration in YAML files and database tables caused drift and complicated Docker/desktop portability.
* **Decision**: Removed `configs/config.yaml`. Bootstrap parameters (proxying, strict MQTT, retention, backup cadence, database driver) load via `LoadDotEnv()` from `.env`, while dynamic operational settings (MQTT broker URL, alert templates, remote tunnels) reside in SQL `settings` with runtime hooks such as `SetOnMQTTConfigChanged` -> `Reconnect(brokerURL)`.

---

## ADR-02: Zero-Trust Out-of-Band Pairing (`POST /api/pair`) & Strict MQTT

* **Context**: Auto-registering devices and returning secrets synchronously in pairing responses exposes infrastructure to device spoofing and queue flooding.
* **Decision**:
  * `POST /api/pair` validates payload size (64 KB), limits `PENDING` queues to 50 devices (`429`), rejects duplicates (`409 Conflict`), and returns `202 Accepted` **without** secrets.
  * An `ADMIN` authorizes the 5-minute TTL `nox_live_` token out-of-band in `/devices`; `POST /api/telemetry` verifies `X-Device-Token` / `Bearer`, `X-Device-Origin`, and `±300s` timestamps.
  * `MONITOR_MQTT_STRICT` defaults to `true`, ignoring unregistered/paused devices on `noxfort/telemetry/#`.

---

## ADR-03: Persistent Hashed Sessions (`DBSessionStore`) & Lockout Protection

* **Context**: In-memory sessions were lost on restarts or DB engine switches; raw tokens and unthrottled logins heightened credential risk.
* **Decision**:
  * Replaced in-memory maps with `DBSessionStore` storing `SHA-256(token)` hashes (`sessions` table, 24h TTL, hourly cleanup), participating in `ReloadableRepository`.
  * Enforced `PBKDF2Hasher` (`100,000` iterations HMAC-SHA256) and `LoginAttemptTracker` (`5` attempts / `15m` / `5m` lockout returning `429`).
  * Rejected `?token=` query authentication and locked down `OPERATOR` accounts strictly to read-only routes + self-service password updates.

---

## ADR-04: Dual-Engine Storage Integrity, Backups & Retention

* **Context**: Partial database migrations or unbounded telemetry/audit growth compromise operational resilience.
* **Decision**:
  * Enabled SQLite WAL mode (`journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout=5000`).
  * Wrapped `Migrator.MigrateData` across all 9 tables in a single database transaction so failed switches roll back cleanly.
  * Added `BackupManager` (`VACUUM INTO`, `pg_dump -F c`, download + `PRAGMA quick_check` / `pg_restore`) and `RetentionManager` (`MONITOR_RETENTION_DAYS=90`).

---

## ADR-05: Native Prometheus Instrumentation & Synchronous Alert Linkage

* **Decision**: Added `internal/metrics` (`/metrics`), exponential-like retry backoff (`500ms`, `1500ms`) + `AlertService.Shutdown(ctx)`, and synchronous `SaveEventReturningID` on state transitions so `alert_dispatch_logs.telemetry_id` preserves exact causal traceability.
