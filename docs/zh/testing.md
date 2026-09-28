# 🧪 测试规范与质量保证 (QA)

本文档规范 **Noxfort Monitor™ v2.0** 的测试体系，涵盖基于内存 Mock 仓储的自动化单元测试，以及通过 MQTT 和 HTTP REST 进行的手动端到端 (E2E) 验证。

⬅️ [文档中心](../README.md) | 🏛️ [系统架构](architecture.md) | 📡 [API 规范](api_reference.md)

---

## 1. 自动化单元测试套件

运行完整测试套件：
```bash
make test
# 底层等效于: go test ./... -v
```

### 内存 Mock 仓储架构
`internal/monitor` 与 `internal/security` 中的核心业务逻辑通过 Mock 接口与物理磁盘彻底解耦：
* `MockDeviceRepository`：模拟设备查询与 `LastSeen` 刷新。
* `MockTelemetryRepository`：验证告警日志的持久化调用。
* `MockAlertDispatcher`：断言事件是否正确送达路由分发策略。
* `MockNotificationChannel`：测试报警渠道逻辑，无需发起真实的 SMTP 或 Telegram 网络请求。

---

## 2. 手动端到端 (E2E) 遥测验证

### 2.1 MQTT 遥测上报测试 (`mosquitto_pub`)

```bash
# 设备保活心跳 (Keep-Alive)
mosquitto_pub -t "noxfort/devices/pump-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# 严重硬件故障告警 (触发即时推送)
mosquitto_pub -t "noxfort/devices/pump-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "CRITICAL",
  "message": "液压管道过压报警: 180 bar",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 HTTP REST 遥测接入测试 (`cURL`)

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "模型推理耗时 4.2ms (预期上限 3.5ms)",
    "occurred_at": "2026-09-27T18:10:00Z"
  }'
```
*预期响应结果*：HTTP `200 OK`，返回 `{"status":"received"}`。
