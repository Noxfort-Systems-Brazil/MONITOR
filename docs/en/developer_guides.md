# Noxfort Monitor — Developer Guides & Conventions

This guide outlines repository conventions, extension interfaces, observability hooks, and build tooling for contributors working on Noxfort Monitor.

---

## 1. Repository Tooling (`Makefile`)

* `make test`: Runs backend Go tests (`go test ./... -v`) and frontend Vitest suite (`npm test`).
* `make test-frontend`: Runs Vitest JavaScript tests (`npm test`).
* `make lint`: Executes Go vet/lint verification across the workspace.
* `make record-demo`: Exercises automated demo workflows via `scripts/demo/` against `--demo` mode.

---

## 2. Persistence & `ReloadableRepository` Contract

Every repository backed by `DBManager` must implement `ReloadableRepository`:

```go
type ReloadableRepository interface {
    SetDB(db *sql.DB, driver string)
}
```

Currently 7 repositories implement this contract (`Device`, `Telemetry`, `Settings`, `Contact`, `Audit`, `User`, and `DBSessionStore`), ensuring dynamic SQLite ↔ PostgreSQL transitions via `Migrator.MigrateData` (single transaction across 9 tables) preserve all active sessions and state. Never rely on ephemeral in-memory session maps or SQLite-only dialect assumptions without parity in PostgreSQL schemas.

---

## 3. HTTP Handler & Security Conventions

* **Page Rendering**: Use `RenderPageWithRequest` so navigation chrome, role context, and CSRF/security attributes render consistently.
* **RBAC Violations**: Use `writeForbiddenResponse` so API/XHR requests receive JSON `403` while browser/desktop HTML views render `web/templates/forbidden.html`.
* **Session & IP Extraction**: Always rely on `ExtractSessionToken` (which intentionally blocks `?token=` query params) and `ExtractClientIP` / `IsHTTPSRequest` (which enforce `TRUSTED_PROXIES` and `MONITOR_USE_HTTPS`).
* **Secret Redaction**: Never render raw API keys (`ApiKey`) or tunnel secrets (`duckdns_token`) in HTML templates on edit/list views.

---

## 4. Telemetry, Metrics, Alerts & Shutdown Lifecycle

* **Metrics (`internal/metrics`)**: Update counters/gauges through `IncTelemetryIngested`, `IncTelemetryRejected`, `IncAlertDispatched`, and `SetBufferDepth` so `GET /metrics` stays accurate.
* **Telemetry-to-Alert Linkage**: Async telemetry uses `BufferedTelemetryWriter`, whereas state transitions triggered by `StateManager`/`Engine` use synchronous `SaveEventReturningID` + `TriggerAlertWithTelemetryID` so `alert_dispatch_logs.telemetry_id` stays linked.
* **Lifecycle (`bgCtx`)**: Pass long-lived background contexts into `RetentionManager`, `BackupManager.StartScheduledBackups`, session cleanup, and MQTT retry/reconnect loops, and drain `AlertService.Shutdown(shutdownCtx)` before closing databases.
