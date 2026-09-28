# 🏛️ Noxfort Monitor™: System Architecture Blueprint

This document specifies the internal architecture of **Noxfort Monitor™ v2.0**, detailing the Event-Driven Architecture (EDA), goroutine concurrency models, SOLID layer boundaries, and the composition root assembly.

⬅️ [Central Hub](../README.md) | 📡 [API Reference](api_reference.md) | 🗄️ [Database](database.md) | 🔐 [Security](security.md) | 🧪 [Testing](testing.md)

---

## 1. Architectural Philosophy

Noxfort Monitor operates on a strict **Event-Driven Architecture (EDA)** coupled with **SOLID** principles and **Dependency Injection (DI)** assembled at the composition root in `cmd/server/main.go`. The system eliminates mutable global state and packages with hidden initialization routines.

```mermaid
graph TD
    subgraph "External World & Edge Nodes"
        LocalDevice[Local Device / LAN]
        RemoteDevice[Remote Agent / WAN (Carina, Synapse)]
        Operator[Browser / Human Operator]
    end

    subgraph "Transport & Network Layer"
        MQTT[MQTT Broker :1883]
        Ngrok[Ngrok Tunnel / WAN HTTPS]
        HTTP[HTTP Server :22100]
        AuthMW[AuthMiddleware - RBAC]
    end

    subgraph "Logic & Security Layer"
        StateManager[State Manager]
        Watchdog[Watchdog Engine]
        Alerts[Alert Routing Service]
        SecManager[Security Manager]
        TunnelMgr[Tunnel Manager]
    end

    subgraph "Dual-Engine Persistence (internal/storage)"
        DBMgr[Central DBManager]
        AuditRepo[AuditRepository]
        PG[(Industrial PostgreSQL)]
        SQLite[(Pure-Go SQLite)]
    end

    subgraph "External Notification Channels"
        Telegram[Telegram Bot API (MarkdownV2)]
        Email[SMTP Server (Email)]
    end

    LocalDevice -- "MQTT Publish" --> MQTT
    RemoteDevice -- "HTTPS POST /api/telemetry" --> Ngrok
    Ngrok --> HTTP
    Operator -- "HTTP GET / POST" --> HTTP

    MQTT -- "Decodes Payload" --> StateManager
    HTTP --> AuthMW
    AuthMW --> StateManager
    AuthMW --> SecManager

    StateManager -- "1. Persists Incident" --> DBMgr
    StateManager -- "2. Dispatches Alert" --> Alerts
    Watchdog -- "Checks Heartbeats" --> DBMgr
    Watchdog -- "Synthesizes Outage/Recovery" --> Alerts
    Watchdog -- "Records Transition" --> AuditRepo

    Alerts -- "Concurrent Goroutines" --> Telegram
    Alerts -- "Concurrent Goroutines" --> Email
    Alerts -- "Records Dispatch SLA" --> AuditRepo

    SecManager -- "Audits Logins" --> AuditRepo
    DBMgr --> PG
    DBMgr --> SQLite
```

---

## 2. Decoupled Subsystem Layers

1. **Transport Layer (`internal/transport`)**: Network protocol termination (MQTT broker via Paho and HTTP REST server with `AuthMiddleware`).
2. **Monitor Logic Layer (`internal/monitor`)**: Reactive state machine (`StateManager`), silent failure detector (`Engine`/Watchdog), and alert dispatcher (`AlertService`).
3. **Security Layer (`internal/security`)**: Session management, salted password hashing, token validation, and Role-Based Access Control (RBAC).
4. **Remote Access Layer (`internal/tunnel`)**: Secure reverse tunneling via Ngrok for WAN-based edge node ingestion.
5. **Domain Layer (`internal/domain`)**: Universal data models and decoupled interface contracts without third-party dependencies.
6. **Persistence Layer (`internal/storage`)**: Dynamic dual-engine manager (`DBManager`), PostgreSQL and SQLite implementations, dialect adapter, and data migrator.
7. **Desktop Interface Layer (`internal/desktop`, `internal/tray`)**: Native runtime in Wails v2 with WebKitGTK, single-instance socket enforcement, and system tray lifecycle.

---

## 3. Concurrency & Goroutine Threading Model

```mermaid
graph TD
    Main[Main OS Thread / Primary Goroutine]
    
    Main -->|Standard GUI Mode| WailsEventLoop[Wails v2 Desktop Event Loop]
    WailsEventLoop --> Systray[Systray GTK Callbacks]
    WailsEventLoop --> WebKit[WebKitGTK Window]
    
    Main -->|--headless Mode| SigChan[Signal Notify Loop (SIGINT/SIGTERM)]

    Main -.->|go func| HTTPServer[HTTP Server ListenAndServe :22100]
    Main -.->|go func| MQTTListener[Paho MQTT Packet Loop]
    Main -.->|go func| WatchdogEngine[Ticker Loop - 30s Interval]
    Main -.->|go func| AlertWorkers[Concurrent Email/Telegram Workers]
```

* **Main Thread**: In desktop mode, executes `desktopApp.Run()` (Wails v2), required because Linux GTK/WebKit toolkits must own the primary OS thread. In `--headless` mode, blocks on an OS signal notification channel (`syscall.SIGTERM`, `os.Interrupt`).
* **HTTP Server**: Runs in an independent goroutine with 15-second read/write timeouts on port `22100`.
* **MQTT Listener**: Paho client manages TCP sockets across dedicated read/write goroutines on port `1883`.
* **Watchdog Loop**: Operates on a decoupled `time.Ticker` channel evaluating silent failure (> 5 minutes).
* **Graceful Shutdown**: Coordinated by a thread-safe `sync.Once` routine releasing all network sockets, tunnels, and database pools cleanly.
