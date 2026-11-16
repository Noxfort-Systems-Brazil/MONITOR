# 🔬 技术预研与架构决策记录 (ADR)：Noxfort Monitor™

本文档记录 **Noxfort Monitor™ v2.0** 的关键架构决策（ADR）与技术演进脉络。

⬅️ [文档中心](../README.md) | 🏛️ [系统架构](architecture.md) | 🗄️ [双引擎数据库](database.md) | 🌐 [远程访问](remote_access.md)

---

## ADR-01：移除 `configs/config.yaml`，全面转向 `.env` + SQL `settings`
* **决策**：移除 `configs/config.yaml` 避免多处配置漂移。启动级环境变量经 `LoadDotEnv()` 从 `.env` 读取，业务运行参数（MQTT Broker URL、告警策略、远程隧道）统一持久化于 SQL `settings`，保存时通过 `SetOnMQTTConfigChanged` $\rightarrow$ `Reconnect` 即时生效。

## ADR-02：Zero-Trust 设备配对 (`POST /api/pair`) 与严格 MQTT 策略
* **决策**：`POST /api/pair` 限制 64 KB 请求体与 50 台 `PENDING` 上限（超限 `429`，已激活返回 `409 Conflict`），并返回 `202` **且不泄露密钥**；管理员在 `/devices` 审批并带外复制 `nox_live_` 密钥（5 分钟 TTL），`POST /api/telemetry` 校验 `X-Device-Token`/`Bearer`、`X-Device-Origin` 及 `±300s` 时间窗；MQTT 启用 `MONITOR_MQTT_STRICT=true` 过滤 `noxfort/telemetry/#`。

## ADR-03：持久化哈希会话 (`DBSessionStore`)、PBKDF2 锁定与严格只读 `OPERATOR`
* **决策**：以持久化 `sessions` 表（`SHA-256` `token_hash`，24h TTL，加入 `ReloadableRepository`）替代易失内存映射，封禁 `?token=` 查询参数；引入 `PBKDF2Hasher`（10万次迭代）与 `LoginAttemptTracker`（5次/15m $\rightarrow$ 5m `429` 锁定），将 `OPERATOR` 严格锁定为只读（外加自助 `POST /api/users/password`）。

## ADR-04：SQLite WAL、单事务 9 表迁移、校验恢复与数据自动保留
* **决策**：基于纯 Go `modernc.org/sqlite` 开启 `WAL` 模式；`Migrator.MigrateData` 在单一事务内迁移 9 张表确保失败回滚；引入 `BackupManager`（`VACUUM INTO`/`pg_dump -F c` + 下载与 `PRAGMA quick_check`/`pg_restore` 恢复）以及 `RetentionManager`（`MONITOR_RETENTION_DAYS=90`）。

## ADR-05：原生 Prometheus 指标、告警重试排空与 `--demo` 模式
* **决策**：新增 `internal/metrics` (`/metrics`)、`SaveEventReturningID` 关联 `alert_dispatch_logs.telemetry_id`、指数退避重试（`500ms`/`1500ms`）+ `AlertService.Shutdown(ctx)` 优雅排空，并提供面向浏览器展示的 `--demo` 模式。
