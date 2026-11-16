# 👨‍💻 高级开发者指南：Noxfort Monitor™

本文档介绍 **Noxfort Monitor™ v2.0** 的工程开发规范：`Makefile` 构建目标、`ReloadableRepository` 仓储热重载契约、HTTP 安全辅助工具、`internal/metrics` 埋点与优雅停机生命周期。

⬅️ [文档中心](../README.md) | 🏛️ [系统架构](architecture.md) | 🧪 [测试与质量控制](testing.md) | 🗄️ [双引擎数据库](database.md)

---

## 1. 系统依赖与核心 `Makefile` 指令

* **Ubuntu / Debian 桌面构建依赖**：Go 1.22+、`libgtk-3-dev`、`libwebkit2gtk-4.1-dev`、`libappindicator3-dev`（或 `libayatana-appindicator3-dev`）。

| 指令 | 描述 |
| :--- | :--- |
| `make build` | 编译桌面二进制文件 (`bin/noxfort-monitor`)。 |
| `make run` | 以包含 Wails v2 GUI 的桌面开发模式运行。 |
| `make run-headless` | 以无界面服务器 (`--headless`) 模式运行。 |
| `make test` | 运行 Go 后端 (`go test ./... -v`) 与 Vitest 前端测试 (`npm test`)。 |
| `make test-frontend` | 仅运行 Vitest JavaScript 单元测试。 |
| `make lint` | 执行 `go vet` 静态代码检查。 |
| `make record-demo` | 针对 `--demo` 模式运行 `scripts/demo/` 演示脚本。 |
| `make deb` / `make install-deb` / `make uninstall-deb` | 打包及安装/卸载 Debian `.deb` 安装包。 |
| `make broker-start` / `make broker-stop` / `make broker-auth` | 管理本地 Mosquitto MQTT 服务与 PBKDF2 认证。 |

---

## 2. 架构约定与持久化契约

* **组合根**：所有依赖均在 [`cmd/server/main.go`](../../cmd/server/main.go) 显式组装，通过 `LoadDotEnv()` 读取 `.env`。
* **`ReloadableRepository` 接口（共 7 个仓储）**：`Device`、`Telemetry`、`Settings`、`Contact`、`Audit`、`User` 与 `DBSessionStore` 均须实现：
  ```go
  type ReloadableRepository interface {
      SetDB(db *sql.DB, driver string)
  }
  ```
  配合 [`storage.AdaptQuery`](../../internal/storage/query_adapter.go) 与单事务 `Migrator.MigrateData`（9 张表）保证 SQLite WAL $\leftrightarrow$ PostgreSQL 无损切换。
* **HTTP 安全与渲染规范**：
  * 统一使用 `RenderPageWithRequest` 与 `writeForbiddenResponse`（API 返回 JSON `403`，UI 返回 [`web/templates/forbidden.html`](../../web/templates/forbidden.html)）。
  * 统一使用 `ExtractSessionToken`（禁止 `?token=`）、`ExtractClientIP` 与 `IsHTTPSRequest`（校验 `TRUSTED_PROXIES` / `MONITOR_USE_HTTPS`）。
  * HTML 渲染时务必脱敏 `ApiKey` 与 `duckdns_token`。
* **指标、告警追溯与生命周期 (`bgCtx`)**：
  * 使用 `internal/metrics` 的 `IncTelemetryIngested`、`IncTelemetryRejected`、`IncAlertDispatched` 与 `SetBufferDepth`。
  * 常规数据走 `BufferedTelemetryWriter`，状态跳变需通过 `SaveEventReturningID` + `TriggerAlertWithTelemetryID` 填充 `alert_dispatch_logs.telemetry_id`；退出时调用 `AlertService.Shutdown(shutdownCtx)`。
