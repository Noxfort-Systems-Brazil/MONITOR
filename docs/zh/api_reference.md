# 📡 API 规范与遥测接入协议

本文档规范 **Noxfort Monitor™ v2.0** 的通信接口，涵盖异步 MQTT 摄取、Zero-Trust HTTP 配对与遥测上报、RBAC 运维管理接口以及 Prometheus 可观测性端点。

⬅️ [文档中心](../README.md) | 🏛️ [系统架构](architecture.md) | 🗄️ [数据库](database.md) | 🔐 [安全规范](security.md)

---

## 1. MQTT 遥测摄取协议

* **Broker 地址**: `tcp://127.0.0.1:1883`（支持在 `/settings` 中热更新并调用 `Reconnect(brokerURL)`，启动失败时每 5 秒后台重试）
* **默认发布主题模式**: `noxfort/telemetry/{identifier}`
* **通配符订阅**: `noxfort/telemetry/#`
* **严格设备策略 (`MONITOR_MQTT_STRICT`)**: 默认 `true`；丢弃未注册或处于 `PAUSED`（维护模式）设备的 MQTT 消息。

### 通用 JSON 报文格式 `IncomingEvent`

```json
{
  "category": "HARDWARE",
  "origin": "sensor-temperature-01",
  "level": "CRITICAL",
  "message": "检测到硬件过热: 温度达 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

* **`category`**: `HARDWARE` 或 `SOFTWARE`（决定告警联系人角色路由）。
* **`origin`**: 设备唯一标识符（如 `carina`、`synapse`、`pump-01`）。
* **`level`**: `INFO`、`WARNING` 或 `CRITICAL`（`INFO` 心跳更新 `last_seen` 且不触发告警噪音；`CRITICAL` 触发同步落库 `SaveEventReturningID` 并关联 `telemetry_id` 发送告警）。
* **`message`**: 可读的事件描述。
* **`occurred_at`**: UTC ISO-8601 时间戳（HTTP 摄取校验 `±300s` 时间偏差窗口）。

---

## 2. Zero-Trust HTTP 设备配对与遥测接入

### 2.1 设备预注册：`POST /api/pair`
* **端口**: `22100`（或经由 Caddy / Ngrok 暴露的 HTTPS 公网入口）
* **认证方式**: 公开接口（请求体上限 `64 KB`，`PENDING` 待审批队列上限为 `50` 台，超出返回 `429 Too Many Requests`）。
* **请求体 JSON**: `{"origin": "synapse-core", "name": "Synapse Core"}`
* **响应说明**:
  * `202 Accepted`: 设备进入 `PENDING` 队列。**响应体不返回任何 API 密钥**（由 `ADMIN` 在 `/devices` 界面完成人工授权并带外复制 5 分钟 TTL 的 `nox_live_` 临时密钥）。
  * `409 Conflict`: 设备已存在且处于 `ACTIVE` 状态。

### 2.2 遥测上报：`POST /api/telemetry`
* **认证方式**: 必须提供 `X-Device-Token: <api_key>` 或 `Authorization: Bearer <api_key>`（常量时间比对）。
* **防伪造与防重放**: 校验 `X-Device-Origin` 请求头与注册设备来源一致，并拒绝 `occurred_at` 超出 `±300s` 窗口的报文。
* **状态响应**: `202 Accepted`（写入 `BufferedTelemetryWriter` 异步队列）、`401 Unauthorized`（令牌缺失/无效/过期）或 `403 Forbidden`（设备处于 `PAUSED` 维护状态或来源不匹配）。

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_your_token" \
  -H "X-Device-Origin: synapse-core" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "数据库连接池占用率超过 85%。",
    "occurred_at": "2026-09-27T18:00:00Z"
  }'
```

---

## 3. 运维与管理 API 端点

受保护接口均依赖持久化 Cookie 会话（`DBSessionStore`），显式拒绝 `?token=` URL 查询令牌，并严格执行 RBAC（`OPERATOR` 为严格只读，仅可自助修改本人密码）。

| 端点 | 方法 | 权限要求 | 说明 |
|---|---|---|---|
| `/api/auth/login` | `POST` | 公开 | 校验凭据并签发 `noxfort_session`；15 分钟内失败 5 次返回 `429`（锁定 5 分钟）。 |
| `/api/auth/logout` | `POST` | 已登录 | 从 `sessions` 表吊销当前会话令牌哈希。 |
| `/api/auth/status` | `GET` | 公开 | 返回认证状态、用户名与角色。 |
| `/api/auth/register` | `POST` | Admin | 管理员注册接口，强制校验密码复杂度。 |
| `/api/dashboard/summary` | `GET` | Operator+ | 实时仪表盘 JSON 摘要（10 秒轮询），涵盖 5 种状态：`ONLINE`、`OFFLINE`、`PENDING`、`PAUSED`、`WAITING`。 |
| `/devices` | `GET` | Operator+ | 渲染设备管理页（页面 HTML 对有效 `ApiKey` 进行脱敏）。 |
| `/devices/create` / `update` / `toggle` | `POST` | Admin | 创建设备（仅一次性明文展示密钥）、重命名或切换维护模式 (`PAUSED`)。 |
| `/devices/rotate-key` / `delete` | `POST` | Admin | 轮换设备 API 密钥或永久删除设备。 |
| `/contacts` 与 `/contacts/{create,update,delete}` | `GET`/`POST` | Op+ / Admin | 查看（`Operator+`）与管理（`Admin`）应急响应联系人。 |
| `/api/users` 与 `/api/users/{create,delete}` | `GET`/`POST` | Admin | 账号管理（禁止删除超级管理员及最后一个管理员）。 |
| `/api/users/password` | `POST` | Operator+ | 用户自助修改本人密码（需验证 `current_password`）。 |
| `/api/users/role` 与 `/api/users/unlock` | `POST` | Admin | 变更 RBAC 角色或解除 `LoginAttemptTracker` 锁定。 |
| `/api/settings/database/status` / `test` / `save` / `provision-user` | `GET`/`POST` | Op+ / Admin | 数据库引擎诊断与 SQLite $\leftrightarrow$ PostgreSQL 单事务热迁移 (`MigrateData`)。 |
| `/api/settings/database/backups` / `backup` | `GET`/`POST` | Admin | 列出或触发热备份快照 (`VACUUM INTO` / `pg_dump -F c`)。 |
| `/api/settings/database/backups/download` / `restore` | `GET`/`POST` | Admin | 安全下载备份（`ResolveBackupPath` 防穿越）与校验恢复（`PRAGMA quick_check` / `pg_restore`）。 |
| `/api/tunnel/{status,save,start,stop,disconnect,test}` | `GET`/`POST` | Op+ / Admin | DuckDNS / Ngrok 状态查看（`Operator+`）与配置控制（`Admin`）。 |
| `/api/audit/security` / `alerts` / `transitions` | `GET` | Operator+ | 查询安全审计、告警发送与状态流转历史。 |
| `/settings/test` 与 `/settings/test-telegram` | `POST` | Admin | 触发 SMTP（支持 465 隐式 TLS / CRLF 过滤）或 Telegram MarkdownV2 测试消息。 |

---

## 4. 公共可观测性与健康诊断接口

| 端点 | 方法 | 功能说明 | 格式 |
|---|---|---|---|
| `/healthz` | `GET` | 存活探针（返回 `200 OK`）。 | `text/plain` |
| `/api/health` | `GET` | 深度健康诊断：数据库状态与延迟、MQTT 连接、内存、Goroutine 与设备状态。 | `application/json` |
| `/metrics` | `GET` | **Prometheus** 指标：`noxfort_uptime_seconds`、`noxfort_goroutines`、`noxfort_memory_heap_alloc_bytes`、`noxfort_memory_heap_sys_bytes`、`noxfort_memory_num_gc`、`noxfort_db_status`、`noxfort_db_latency_ms`、`noxfort_mqtt_connected`、`noxfort_devices_total`、`noxfort_devices_by_status`、`noxfort_telemetry_buffer_depth`、`noxfort_telemetry_ingested_total`、`noxfort_telemetry_rejected_total`、`noxfort_alerts_dispatched_total`。 | `text/plain; version=0.0.4` |
