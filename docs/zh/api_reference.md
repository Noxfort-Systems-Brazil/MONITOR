# 📡 API 规范与遥测接入协议

本文档正式规范 **Noxfort Monitor™ v2.0** 的外部通信接口，包括异步 MQTT 摄取协议、HTTP REST 遥测接入端点、运维管理接口以及系统可观测性与健康检查接口。

⬅️ [文档中心](README.md) | 🏛️ [系统架构](architecture.md) | 🗄️ [数据库](database.md) | 🔐 [安全规范](security.md)

---

## 1. MQTT 遥测摄取协议

* **Broker 地址**: `tcp://127.0.0.1:1883`
* **默认设备主题模式**: `noxfort/devices/{identifier}/telemetry`
* **通配符订阅**: `noxfort/devices/+/telemetry`

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

#### 字段规范说明：
* **`category`** (*String*): `HARDWARE` 或 `SOFTWARE`。决定 RBAC 报警路由策略（硬件 $\rightarrow$ 技术员；软件 $\rightarrow$ 程序员）。
* **`origin`** (*String*): 设备唯一标识符（例如 `carina`, `synapse`, `pump-01`）。
* **`level`** (*String*): `INFO`, `WARNING` 或 `CRITICAL`。
  * 包含保活关键字 ("*system ok*", "*heartbeat*", "*online*") 的 `INFO` 消息仅更新设备心跳时间戳，避免告警噪音和数据库写入开销。
  * `CRITICAL` 级别消息将立即触发多渠道报警推送。
* **`message`** (*String*): 易读的事件描述文本。
* **`occurred_at`** (*String ISO-8601*): UTC 时间戳。

---

## 2. 基于 HTTP REST 的遥测接入

### `POST /api/telemetry`
专为外部智能体（如 **Carina**, **Synapse**、物联网边缘网关或 cURL 脚本）设计。

* **端口**: `22100`（或 Caddy / Ngrok 公网域名）
* **权限验证**: 公开接口（无需 Token，适配边缘自主传感器）。
* **网络隔离**: 在生产环境 (`ExternalIngestionHandler`) 下，端口 `22100` 仅对外开放 `POST /api/telemetry`、`/healthz`、`/api/health` 及 `/metrics`。浏览器直接访问控制台页面将被拦截并返回 **403 Forbidden**。
* **请求头**: `Content-Type: application/json`
* **请求体**: 与 `IncomingEvent` 结构一致。

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
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

*可在原生桌面应用程序中直接操作，或通过经过身份验证的 HTTP 会话访问。*

| 端点 | 请求方法 | 权限要求 | 功能说明 |
|---|---|---|---|
| `/api/auth/login` | `POST` | 公开 | 校验用户名密码并签发 `noxfort_session` Cookie。 |
| `/api/auth/logout` | `POST` | 已登录 | 注销当前有效会话。 |
| `/api/auth/status` | `GET` | 公开 | 返回会话验证状态、用户名与角色。 |
| `/api/auth/register` | `POST` | Admin | 经过鉴权的创建账号别名，强制执行工业级密码复杂度校验。 |
| `/devices` | `GET` | Operator+ | 渲染设备管理后台页面，展示已注册节点与 `LastSeen` 状态。 |
| `/devices/delete` | `POST` | Admin | 通过 POST 表单参数 (`id=xyz`) 删除设备并重定向 (`303`)。 |
| `/contacts` | `GET` | Operator+ | 渲染应急响应团队管理页面。 |
| `/contacts/create` | `POST` | Admin | 登记新告警联系人（姓名、邮箱、电话、Telegram ID、角色）。 |
| `/contacts/update` | `POST` | Admin | 更新联系人信息及通知渠道。 |
| `/contacts/delete` | `POST` | Admin | 从应急响应团队中移除指定联系人。 |
| `/api/users` | `GET` | Admin | 返回包含所有已注册操作员的 JSON 数组。 |
| `/api/users/create` | `POST` | Admin | 创建操作员/管理员账号（校验密码复杂度）。 |
| `/api/users/delete` | `POST` | Admin | 删除操作员账号（默认超级用户受保护不可删除）。 |
| `/api/settings/database/status` | `GET` | 已登录 | 返回当前活动数据库引擎、延迟和表结构状态。 |
| `/api/settings/database/test` | `POST` | Admin | 切换前测试目标 PostgreSQL 数据库连接。 |
| `/api/settings/database/save` | `POST` | Admin | 运行时热切换仓储连接至目标引擎（支持 `migrate=true`）。 |
| `/api/settings/database/provision-user` | `POST` | Admin | 在 PostgreSQL 中自动创建受限权限的专用业务账号。 |
| `/api/settings/database/backup` | `POST` | Admin | 触发非阻塞数据库热备份快照 (`VACUUM INTO` / `pg_dump`)。 |
| `/api/settings/database/backups` | `GET` | Admin | 列出 `backups/` 目录下所有可用的备份文件。 |
| `/api/tunnel/status` | `GET` | 已登录 | 返回 WAN 连接状态、提供商 (DuckDNS/Ngrok)、公网 URL 与 IPv6。 |
| `/api/tunnel/save` | `POST` | Admin | 将 DuckDNS / Ngrok 参数持久化至数据库。 |
| `/api/tunnel/start` | `POST` | Admin | 触发动态 DNS 同步或启动反向隧道。 |
| `/api/tunnel/stop` | `POST` | Admin | 停止 WAN 同步并禁用开机自启。 |
| `/api/tunnel/disconnect` | `POST` | Admin | 从数据库解绑凭证并关闭服务。 |
| `/api/tunnel/test` | `POST` | Admin | 实时验证 DuckDNS 令牌与域名解析。 |
| `/api/audit/security` | `GET` | Admin | 查询系统安全与管理审计日志。 |
| `/api/audit/alerts` | `GET` | Admin | 查询报警分发日志与 SLA 履约记录。 |
| `/api/audit/transitions` | `GET` | Admin | 查询设备故障/恢复状态流转与停机时长。 |
| `/settings/test` | `POST` | Admin | 使用当前 SMTP 配置发送测试邮件。 |
| `/settings/test-telegram` | `POST` | Admin | 向配置的 Telegram Bot 发送 MarkdownV2 测试告警。 |
| `/api/open-external` | `POST` | 已登录 | 命令操作系统在默认浏览器中打开指定 URL。 |
| `/api/window/toggle-fullscreen` | `POST` | 公开 | 在全屏与常规窗口之间切换 Wails 桌面。 |
| `/api/window/exit-fullscreen` | `POST` | 公开 | 退出全屏模式。 |

---

## 4. 公共可观测性与健康诊断接口

*无需会话验证，适用于 Docker/K8s 探针和 Prometheus 抓取。*

| 端点 | 请求方法 | 功能说明 | 返回格式 |
|---|---|---|---|
| `/healthz` | `GET` | 存活探针（Liveness Probe），返回 `200 OK` (`"OK\n"`)。 | `text/plain` |
| `/api/health` | `GET` | 深度诊断探针：数据库状态与延迟、MQTT 连通性、内存分配、Goroutine 数量及设备数。 | `application/json` |
| `/metrics` | `GET` | **Prometheus** 指标导出 (`noxfort_uptime_seconds`, `noxfort_goroutines`, `noxfort_db_status`, `noxfort_mqtt_connected`)。 | `text/plain; version=0.0.4` |
