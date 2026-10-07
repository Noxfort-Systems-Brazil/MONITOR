# 👨‍💻 高级开发者指南：Noxfort Monitor™

本文档为开发、维护和扩展 **Noxfort Monitor™ v2.0** 提供了软件工程规范与技术指南，涵盖本地开发环境搭建、**SOLID** 架构约定、支持**热重载**的仓储模式（Repository Pattern）、依赖注入以及并发设计模型。

⬅️ [文档中心](README.md) | 🏛️ [系统架构](architecture.md) | 🧪 [测试与质量控制](testing.md) | 🗄️ [双引擎数据库](database.md)

---

## 1. 本地开发环境与编译构建

### 1.1 系统依赖要求 (Ubuntu / Debian)
* **Go 工具链**：版本 1.22 或更高。
* **C/GTK 与 WebKit 库**：Wails v2 桌面运行时与系统托盘必须依赖：
  ```bash
  sudo apt-get update
  sudo apt-get install -y \
      libgtk-3-dev \
      libwebkit2gtk-4.1-dev \
      libappindicator3-dev || sudo apt-get install -y libayatana-appindicator3-dev
  ```
* **本地 Mosquitto Broker**：
  ```bash
  make broker-install
  make broker-auth
  make broker-start
  ```

### 1.2 核心 Makefile 指令

| 指令 | 描述 |
| :--- | :--- |
| `make build` | 使用生产标签编译 Wails 桌面二进制文件 (`bin/noxfort-monitor`)。 |
| `make run` | 以包含图形界面的桌面模式启动应用。 |
| `make run-headless` | 以无界面服务器（Headless）模式运行。 |
| `make test` | 运行自动化测试套件（Go 后端测试 + Vitest 前端测试）。 |
| `make deb` | 构建 Debian 分发安装包 (`noxfort-monitor_2.0.1_amd64.deb`)。 |
| `make install-deb` | 通过 `apt` 安装或更新生成的 `.deb` 安装包。 |
| `make uninstall-deb` | 卸载 `noxfort-monitor` 系统包。 |
| `make broker-start` | 启动本地 Mosquitto MQTT 服务。 |
| `make broker-stop` | 停止本地 MQTT 服务。 |
| `make broker-status`| 查看 Mosquitto 服务运行状态。 |

---

## 2. 设计模式与架构约定

Noxfort Monitor 在事件驱动分层架构中严格贯彻 **SOLID** 设计原则。

### 2.1 依赖组装根节点 (Composition Root)
所有的对象实例化与依赖注入均集中且显式地在 [`cmd/server/main.go`](../../cmd/server/main.go) 中完成。
* **零全局可变状态**：任何包和模块均不得维护可变的全局变量，严禁依赖不可预测的 `init()` 副作用函数。
* **依赖注入 (DI)**：所有服务构造函数均通过接收接口实现依赖注入，保证模块间彻底解耦，并支持单元测试中使用内存 Mock。

### 2.2 支持热重载的仓储模式 (Repository Pattern)
[`internal/storage`](../../internal/storage) 持久层通过 [`internal/domain`](../../internal/domain) 中定义的接口隔离具体存储机制：
* 仓储均实现 `ReloadableRepository` 接口：
  ```go
  type ReloadableRepository interface {
      SetDB(db *sql.DB, driver string)
  }
  ```
* 当管理员在运行时动态切换 SQLite 与 PostgreSQL 时，[`DBManager`](../../internal/storage/db_manager.go) 通过调用 `SetDB` 自动热更新所有已注册的仓储连接。

### 2.3 并发最佳实践
* **看门狗状态追踪**：设备在线心跳状态由 [`SystemStatusTracker`](../../internal/monitor/tracker.go) 在内存中维护，并通过 `sync.RWMutex` 严格保护，彻底杜绝数据竞态。
* **异步告警分发**：通过邮件（SMTP）与 Telegram 发送的通知在 [`AlertService.TriggerAlert()`](../../internal/monitor/alerts.go) 中作为独立的 Goroutine 并发派发，网络延迟绝不阻塞 MQTT 主事件流。

---

## 3. 系统扩展开发

### 3.1 添加新通知渠道（如 Discord、Slack 或 企业微信）
1. 实现 [`internal/monitor/channel.go`](../../internal/monitor/channel.go) 中定义的 `NotificationChannel` 接口：
   ```go
   type NotificationChannel interface {
       Name() string
       Send(settings *domain.Settings, contact *domain.Contact, identifier string, event *domain.IncomingEvent) error
       Recipient(contact *domain.Contact) string
   }
   ```
2. 在 [`Settings`](../../internal/domain/settings.go) 结构体中添加必要的配置字段。
3. 在 `cmd/server/main.go` 中的 `AlertService` 构造函数注册新渠道。

### 3.2 添加新数据表或业务实体
1. 在 `internal/domain` 中定义结构体实体与对应的 Repository 接口。
2. 在 `internal/storage` 中实现该仓储，实现 `ReloadableRepository`，并使用 [`storage.AdaptQuery`](../../internal/storage/query_adapter.go) 抹平 SQLite 与 PostgreSQL 的 SQL 语法差异。
3. 在 `database.go` 的 `NewDatabase` 中添加 SQLite 建表语句，并在 `postgres_schema.go` 的 `InitPostgresSchema` 中添加 PostgreSQL 建表语句。
4. 如需数据迁移支持，在 `migrator.go` 中注册相应迁移逻辑。
