# 🏛️ Noxfort Monitor™: 系统架构与并发模型设计

本文档详细规范 **Noxfort Monitor™ v2.0** 的内部技术架构，深入剖析事件驱动架构 (EDA)、Goroutine 并发调度模型、SOLID 分层边界及组合根组装机制。

⬅️ [文档中心](../README.md) | 📡 [API 规范](api_reference.md) | 🗄️ [数据库](database.md) | 🔐 [安全规范](security.md) | 🧪 [测试指南](testing.md)

---

## 1. 宏观架构设计哲学

Noxfort Monitor 遵循严格的**事件驱动架构 (EDA)**，融合 **SOLID** 设计原则与**依赖注入 (DI)**，统一在 `cmd/server/main.go` 组合根中组装，杜绝共享全局可变状态及隐式初始化的包。

```mermaid
graph TD
    subgraph "外部世界与边缘节点"
        LocalDevice[本地设备 / 局域网]
        RemoteDevice[远程智能体 / 广域网 (Carina, Synapse)]
        Operator[浏览器 / 运维人员]
    end

    subgraph "网络传输层"
        MQTT[MQTT Broker :1883]
        Ngrok[Ngrok 隧道 / 广域网 HTTPS]
        HTTP[HTTP 服务器 :22100]
        AuthMW[AuthMiddleware - RBAC]
    end

    subgraph "业务逻辑与安全层"
        StateManager[状态管理器]
        Watchdog[看门狗引擎]
        Alerts[报警分发路由服务]
        SecManager[安全管理器]
        TunnelMgr[隧道管理器]
    end

    subgraph "双引擎持久化层 (internal/storage)"
        DBMgr[中央 DBManager]
        AuditRepo[AuditRepository]
        PG[(企业级 PostgreSQL)]
        SQLite[(纯 Go 嵌入式 SQLite)]
    end

    subgraph "外部通知信道"
        Telegram[Telegram Bot API (MarkdownV2)]
        Email[SMTP 邮件服务器]
    end

    LocalDevice -- "MQTT 发布" --> MQTT
    RemoteDevice -- "HTTPS POST /api/telemetry" --> Ngrok
    Ngrok --> HTTP
    Operator -- "HTTP GET / POST" --> HTTP

    MQTT -- "解析报文" --> StateManager
    HTTP --> AuthMW
    AuthMW --> StateManager
    AuthMW --> SecManager

    StateManager -- "1. 写入事件" --> DBMgr
    StateManager -- "2. 触发报警" --> Alerts
    Watchdog -- "轮询心跳" --> DBMgr
    Watchdog -- "合成故障/恢复事件" --> Alerts
    Watchdog -- "记录状态跳变" --> AuditRepo

    Alerts -- "并发 Goroutine" --> Telegram
    Alerts -- "并发 Goroutine" --> Email
    Alerts -- "记录推送 SLA" --> AuditRepo

    SecManager -- "审计登录行为" --> AuditRepo
    DBMgr --> PG
    DBMgr --> SQLite
```

---

## 2. 解耦子系统分层

1. **传输层 (`internal/transport`)**：处理网络通信协议终止（基于 Paho 的 MQTT 客户端及带 `AuthMiddleware` 的 HTTP REST 服务）。
2. **监控核心逻辑层 (`internal/monitor`)**：包含状态流转机 (`StateManager`)、静默故障检测引擎 (`Engine`/Watchdog) 及报警路由器 (`AlertService`)。
3. **安全层 (`internal/security`)**：管理身份验证、加盐密码散列、会话有效性及基于角色的权限控制 (RBAC)。
4. **远程访问层 (`internal/tunnel`)**：管理基于 Ngrok 的安全反向隧道，支持边缘节点穿透内网。
5. **领域核心层 (`internal/domain`)**：定义核心实体与接口契约，不引入任何外部第三方依赖。
6. **持久化层 (`internal/storage`)**：包含运行时热切换管理器 (`DBManager`)、PostgreSQL/SQLite 双驱动、SQL 方言转换器及异构数据迁移器。
7. **桌面交互层 (`internal/desktop`, `internal/tray`)**：基于 Wails v2 与 WebKitGTK，提供 Unix IPC 套接字单实例锁及系统托盘常驻。

---

## 3. 并发调度与 Goroutine 线程模型

```mermaid
graph TD
    Main[主操作系统线程 / 根 Goroutine]
    
    Main -->|标准桌面模式| WailsEventLoop[Wails v2 桌面事件循环]
    WailsEventLoop --> Systray[GTK 系统托盘回调]
    WailsEventLoop --> WebKit[WebKitGTK 渲染窗口]
    
    Main -->|--headless 无头模式| SigChan[操作系统信号监听循环 (SIGINT/SIGTERM)]

    Main -.->|go func| HTTPServer[HTTP 服务器 ListenAndServe :22100]
    Main -.->|go func| MQTTListener[Paho MQTT 报文收发循环]
    Main -.->|go func| WatchdogEngine[Ticker 定时循环 - 30秒间隔]
    Main -.->|go func| AlertWorkers[并发邮件/Telegram 推送协程]
```

* **主操作系统线程**：在桌面模式下驱动 `desktopApp.Run()` (Wails v2)。在 `--headless` 模式下阻塞监听操作系统中断信号 (`syscall.SIGTERM`, `os.Interrupt`)。
* **HTTP 服务器**：独立运行在 Goroutine 中，默认监听 `22100` 端口，具有 15 秒超时保护。
* **MQTT 监听器**：Paho 客户端在独立的 Goroutine 中处理 `1883` 端口上的 TCP 报文。
* **看门狗轮询**：通过 `time.Ticker` 定时器独立运行，检测设备静默状态（超时阈值为 5 分钟）。
* **优雅停机机制**：由线程安全的 `sync.Once` 控制，依次释放网络套接字、关闭反向隧道并安全释放数据库连接池。
