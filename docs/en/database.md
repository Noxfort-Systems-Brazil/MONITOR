# Noxfort Monitor — Database Architecture & Lifecycle

Noxfort Monitor uses a dual-engine persistence layer supporting zero-restart hot switching between embedded **SQLite (WAL mode)** and **PostgreSQL**, backed by transactional migration, hot backup/restore, and automated data retention.

---

## 1. Dual-Engine Architecture & Hot Reload

```
+---------------------------------------------------------------+
|                         DBManager                             |
|   - Holds active *sql.DB handle + dialect ("sqlite"/"postgres")|
|   - Coordinates MigrateData + Switch()                        |
+------------------------------+--------------------------------+
                               | SetDB(db *sql.DB, driver string)
                               v
   +--------------------------------------------------------+
   | ReloadableRepository (7 Repositories)                  |
   |  1. DeviceRepository     5. AuditRepository            |
   |  2. TelemetryRepository  6. UserRepository             |
   |  3. SettingsRepository   7. DBSessionStore             |
   |  4. ContactRepository                                  |
   +--------------------------------------------------------+
```

* **SQLite Concurrency Optimization**: Initialized with `journal_mode=WAL`, `synchronous=NORMAL`, and `busy_timeout=5000` for lock-free concurrent reads and buffered writes.
* **Schema Parity (`9` Tables, `12` Indexes)**: Both SQLite and PostgreSQL maintain identical logical schemas across `devices` (including `api_key`, `status`, `expires_at`), `telemetry` (referencing `device_id`), `settings`, `contacts`, `users`, `sessions` (`token_hash`, `expires_at`), `security_audit_logs`, `alert_dispatch_logs`, and `device_state_transitions`.
* **Atomic Engine Switching**: Calling `SetDB(db *sql.DB, driver string)` dynamically rebinds all 7 `ReloadableRepository` instances at runtime without restarting the HTTP or MQTT servers.

---

## 2. Transactional Live Migration (`MigrateData`)

* When switching from SQLite to PostgreSQL (or vice versa), `Migrator.MigrateData` transfers all **9 tables** inside a **single database transaction**.
* If any table fails to copy or verify, the transaction rolls back cleanly and `DBManager.Switch` aborts, preventing split-brain or partially migrated states.

---

## 3. Hot Backups & Verified Restore (`BackupManager`)

* **Non-Blocking Hot Backups**:
  * **SQLite**: Uses `VACUUM INTO` to produce an atomic, consistent snapshot without stopping telemetry ingestion.
  * **PostgreSQL**: Invokes `pg_dump -F c` custom-format dumps.
* **Scheduled Automation**: `BackupManager.StartScheduledBackups` runs periodically according to `MONITOR_BACKUP_INTERVAL_HOURS` (default `24h`) and retains the most recent **`7`** snapshots.
* **Secure Download & Restore**:
  * `GET /api/settings/database/backups/download` enforces `ResolveBackupPath` to block path traversal.
  * `POST /api/settings/database/restore` validates SQLite uploads with `PRAGMA quick_check` before swapping via `RestoreSQLiteFile`, or restores PostgreSQL archives with `pg_restore`.

---

## 4. Automated Data Retention (`RetentionManager`)

* Controlled by `MONITOR_RETENTION_DAYS` (default **`90` days**), running every **`24` hours**.
* Automatically prunes aged historical records across 4 high-growth tables while preserving registered devices, users, active sessions, and configuration:
  1. `telemetry`
  2. `alert_dispatch_logs`
  3. `security_audit_logs`
  4. `device_state_transitions`
