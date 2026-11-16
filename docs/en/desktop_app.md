# Noxfort Monitor — Desktop Application & Execution Modes

Noxfort Monitor supports embedded native desktop operation (Wails) alongside headless server and live presentation modes from the unified `cmd/server/main.go` entry point.

---

## 1. Execution Modes

* **Desktop GUI Mode (Default)**: Launches the HTTP/MQTT core alongside an embedded Wails webview and system tray controls (`internal/desktop/app.go`).
* **Demo Mode (`--demo`)**: Starts `RunDesktopServer()` directly without Wails window chrome, designed for browser-based walkthroughs and automated demo recordings (`make record-demo`).
* **Headless Server (`--headless` / `--server-only`)**: Disables desktop UI libraries for Linux systemd containers, Docker Compose deployments, and edge gateways.

---

## 2. Desktop UI Capabilities & Security Behavior

* **Real-Time Dashboard Polling**: The dashboard queries `GET /api/dashboard/summary` every **10 seconds**, rendering counts and device rows across all 5 operational states: `ONLINE`, `OFFLINE`, `PENDING`, `PAUSED`, and `WAITING`.
* **In-App `403 Forbidden` Handling**: Read-only `OPERATOR` users attempting to open admin-only desktop pages (e.g., `/users`) or perform restricted HTML actions are cleanly presented with `web/templates/forbidden.html` inside the app frame rather than raw JSON errors.
* **Self-Service Password Modal**: All authenticated users—including `OPERATOR`—can update their own password via `POST /api/users/password` (verifying `current_password`).
* **Credential Redaction & One-Time Keys**: Newly created/paired devices reveal their API key once in `/devices`; subsequent page loads redact `ApiKey` secrets in HTML (`POST /devices/rotate-key` is used for rotation). `/settings` and `/remote` likewise mask sensitive tokens.

---

## 3. Clean Exit & Alert Drain

* Closing the desktop window or sending `SIGINT`/`SIGTERM` cancels background loops (`RetentionManager`, scheduled backups, session cleanup) and invokes `AlertService.Shutdown(shutdownCtx)` so pending Telegram, Email, or Webhook notifications drain before database handles close.
