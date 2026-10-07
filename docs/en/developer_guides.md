# 👨‍💻 Advanced Developer Guide: Noxfort Monitor™

This document provides software engineering guidelines for building, maintaining, and extending **Noxfort Monitor™ v2.0**, covering local setup, **SOLID** architectural conventions, **hot-reloadable** repository patterns, dependency injection, and concurrency models.

⬅️ [Central Hub](README.md) | 🏛️ [Architecture](architecture.md) | 🧪 [Testing & QA](testing.md) | 🗄️ [Database](database.md)

---

## 1. Local Environment & Compilation

### 1.1 System Prerequisites (Ubuntu / Debian)
* **Go Toolkit**: Version 1.22 or higher.
* **C/GTK & WebKit Libraries**: Required for the Wails v2 desktop runtime and system tray:
  ```bash
  sudo apt-get update
  sudo apt-get install -y \
      libgtk-3-dev \
      libwebkit2gtk-4.1-dev \
      libappindicator3-dev || sudo apt-get install -y libayatana-appindicator3-dev
  ```
* **Local Mosquitto Broker**:
  ```bash
  make broker-install
  make broker-auth
  make broker-start
  ```

### 1.2 Essential Makefile Commands

| Command | Description |
| :--- | :--- |
| `make build` | Compiles the Wails desktop binary with production tags (`bin/noxfort-monitor`). |
| `make run` | Runs the full application in development mode with graphical desktop interface. |
| `make run-headless` | Runs the server in Headless mode (without opening desktop windows). |
| `make test` | Executes the complete automated test suite (Go backend + Vitest frontend). |
| `make deb` | Builds the Debian distribution package (`noxfort-monitor_2.0.1_amd64.deb`). |
| `make install-deb` | Installs or upgrades the generated `.deb` package via `apt`. |
| `make uninstall-deb` | Uninstalls the `noxfort-monitor` system package. |
| `make broker-start` | Starts the native Mosquitto MQTT broker. |
| `make broker-stop` | Stops the MQTT broker. |
| `make broker-status`| Checks Mosquitto service status. |

---

## 2. Design Patterns & Architecture

Noxfort Monitor strictly implements **SOLID** design principles within an event-driven layered architecture.

### 2.1 Composition Root
All object instantiation and dependency wiring occurs explicitly in [`cmd/server/main.go`](../../cmd/server/main.go).
* **Zero Global Variables**: No packages or modules maintain mutable global state or rely on magical `init()` routines.
* **Dependency Injection (DI)**: Constructors accept interfaces, guaranteeing decoupling and straightforward unit testing with in-memory mocks.

### 2.2 Repository Pattern with Hot-Reload
The persistence layer in [`internal/storage`](../../internal/storage) isolates storage mechanisms through interfaces defined in [`internal/domain`](../../internal/domain):
* Repositories implement `ReloadableRepository`:
  ```go
  type ReloadableRepository interface {
      SetDB(db *sql.DB, driver string)
  }
  ```
* When an operator switches between SQLite and PostgreSQL, [`DBManager`](../../internal/storage/db_manager.go) notifies all registered repositories via `SetDB`.

### 2.3 Concurrency Best Practices
* **Watchdog State Tracking**: Equipment presence is tracked in memory via [`SystemStatusTracker`](../../internal/monitor/tracker.go), protected by a `sync.RWMutex` to eliminate race conditions.
* **Asynchronous Alert Dispatch**: Notifications dispatched via Email (SMTP) and Telegram run in independent goroutines inside [`AlertService.TriggerAlert()`](../../internal/monitor/alerts.go). Network latency never blocks the MQTT event processing pipeline.

---

## 3. Extending the System

### 3.1 Adding a New Notification Channel (e.g., Discord, Slack, or WhatsApp)
1. Implement the `NotificationChannel` interface defined in [`internal/monitor/channel.go`](../../internal/monitor/channel.go):
   ```go
   type NotificationChannel interface {
       Name() string
       Send(settings *domain.Settings, contact *domain.Contact, identifier string, event *domain.IncomingEvent) error
       Recipient(contact *domain.Contact) string
   }
   ```
2. Add necessary configuration fields to the [`Settings`](../../internal/domain/settings.go) entity.
3. Wire the new channel into the `AlertService` constructor in `cmd/server/main.go`.

### 3.2 Adding a New Table or Entity
1. Define the struct and repository interface in `internal/domain`.
2. Implement the repository in `internal/storage`, implementing `ReloadableRepository` and using [`storage.AdaptQuery`](../../internal/storage/query_adapter.go) to support SQLite and PostgreSQL dialects concurrently.
3. Add the table creation statement to `NewDatabase` in `database.go` (for SQLite) and `InitPostgresSchema` in `postgres_schema.go` (for PostgreSQL).
4. Add a migration routine in `migrator.go` if data preservation is required.
