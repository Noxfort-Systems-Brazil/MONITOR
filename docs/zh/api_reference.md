# 📡 API 规范与遥测接入协议

本文档正式规范 **Noxfort Monitor™ v2.0** 的外部通信接口，包括异步 MQTT 摄取协议、HTTP REST 遥测接入端点及管理接口。

⬅️ [文档中心](../README.md) | 🏛️ [系统架构](architecture.md) | 🗄️ [数据库](database.md) | 🔐 [安全规范](security.md)

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

* **端口**: `22100`（或 Ngrok 公网域名）
* **权限验证**: 公开接口（无需 Token，适配边缘自主传感器）。
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

| 端点 | 请求方法 | 权限要求 | 功能说明 |
|---|---|---|---|
| `/api/auth/login` | `POST` | 公开 | 校验用户名密码并签发 `noxfort_session` Cookie。 |
| `/api/auth/logout` | `POST` | 已登录 | 注销当前有效会话。 |
| `/api/devices` | `GET` | Operator+ | 查询已注册设备列表及 `LastSeen` 状态。 |
| `/api/devices/register` | `POST` | Admin | 注册或更新设备信息。 |
| `/api/settings/database/test` | `POST` | Admin | 测试目标 PostgreSQL 数据库连接。 |
| `/api/settings/database/migrate`| `POST` | Admin | 在 SQLite 与 PostgreSQL 之间执行数据表结构与数据迁移。 |
| `/api/settings/database/save` | `POST` | Admin | 运行时热切换仓储连接至目标引擎。 |
| `/api/tunnel/start` | `POST` | Admin | 启动 Ngrok 反向穿透隧道。 |
| `/api/tunnel/stop` | `POST` | Admin | 关闭反向隧道。 |
| `/api/audit/security` | `GET` | Admin | 查询系统安全审计与认证日志。 |
| `/api/audit/alerts` | `GET` | Admin | 查询报警分发日志与 SLA 履约记录。 |
