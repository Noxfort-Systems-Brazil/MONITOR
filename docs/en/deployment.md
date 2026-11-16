# Noxfort Monitor — Deployment & Operations Guide

This guide covers environment configuration, Docker Compose orchestration, reverse proxying, automated backups, and production observability.

---

## 1. Environment Configuration (`.env`)

Legacy `configs/config.yaml` has been removed; runtime environment variables are loaded from `.env` via `LoadDotEnv()` (see `.env.example`), while dynamic operational settings (MQTT broker URL, alerts, tunnels) persist in the `settings` SQL table.

| Variable | Default | Description |
| :--- | :--- | :--- |
| `MONITOR_MQTT_STRICT` | `true` | Drops MQTT telemetry from unregistered or `PAUSED` devices (`noxfort/telemetry/#`). |
| `TRUSTED_PROXIES` | *(empty)* | Comma-separated CIDRs/IPs trusted for `X-Real-IP`, `X-Forwarded-For`, and `X-Forwarded-Proto`. |
| `MONITOR_USE_HTTPS` | `false` | Forces `Secure` cookies and HTTPS awareness behind TLS terminators. |
| `MONITOR_RETENTION_DAYS` | `90` | Days before `RetentionManager` prunes `telemetry` and audit/transition tables (every 24h). |
| `MONITOR_BACKUP_INTERVAL_HOURS` | `24` | Interval for automated scheduled hot backups (`BackupManager`, retains 7 snapshots). |
| `MONITOR_DB_DRIVER` / `MONITOR_DB_*` | `sqlite` | Optional bootstrap configuration when deploying directly against PostgreSQL. |

---

## 2. Docker & Headless Deployment

```bash
cp .env.example .env
docker compose up -d --build
```

* Runs the server in `--headless` mode alongside Eclipse Mosquitto (`mosquitto/config/mosquitto.conf`) and optional PostgreSQL services (`docker-compose.yml`).
* Run `./scripts/setup_mqtt_auth.sh` to provision Mosquitto credentials before exposing MQTT on a network.

---

## 3. NGINX Reverse Proxy Snippet

When using NGINX instead of `caddy/Caddyfile`, forward proxy headers for trusted IPs and preserve zero-trust `/api/pair` and `/api/telemetry` headers:

```nginx
location / {
    proxy_pass http://127.0.0.1:22100;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Ensure NGINX's IP is listed in `TRUSTED_PROXIES` so `LoginAttemptTracker` and `SecurityAuditLog` see real client IPs instead of the proxy address.

---

## 4. Backups, Restore, Retention & Metrics

* **Scheduled & On-Demand Backups**: `BackupManager` writes atomic snapshots (`VACUUM INTO` for SQLite; `pg_dump -F c` for PostgreSQL), retains up to 7 files automatically, and supports `/api/settings/database/backups/download` + `/api/settings/database/restore` (`PRAGMA quick_check` / `pg_restore`).
* **Retention Pruning**: `RetentionManager` clears aged rows from `telemetry`, `alert_dispatch_logs`, `security_audit_logs`, and `device_state_transitions` every 24h.
* **Prometheus Scraping**: Point Prometheus at `GET /metrics` to collect uptime, goroutines, heap memory, `noxfort_db_status`, `noxfort_db_latency_ms`, `noxfort_mqtt_connected`, `noxfort_devices_total`, `noxfort_devices_by_status`, `noxfort_telemetry_buffer_depth`, `noxfort_telemetry_ingested_total`, `noxfort_telemetry_rejected_total`, and `noxfort_alerts_dispatched_total`.
