# 🧪 测试规范与质量保证 (QA)

本文档规范 **Noxfort Monitor™ v2.0** 的测试体系：自动化单元与集成测试（Go + Vitest）、MQTT (`noxfort/telemetry/#`) 与 Zero-Trust HTTP (`/api/pair` + `/api/telemetry`) 端到端测试以及演示模式录制验证。

⬅️ [文档中心](../README.md) | 🏛️ [系统架构](architecture.md) | 📡 [API 规范](api_reference.md)

---

## 1. 自动化测试套件

```bash
# 运行全部后端 Go 测试 (go test ./... -v) 与前端 Vitest 测试 (npm test)
make test

# 仅运行前端测试 (Vitest: 4 个测试文件，26 个用例；替代原 pytest 流程)
make test-frontend

# 静态检查 (go vet)
make lint
```

* **后端 Go 测试**：覆盖安全模块（`lockout_test.go`、PBKDF2 散列、复杂度）、存储模块（`session_repo_test.go`、`retention_test.go`、事务迁移、备份/恢复）、HTTP RBAC/摄取接口、严格 MQTT、协议解码以及告警（`alerts_test.go`）。
* **内存 Mock 接口**：`MockDeviceRepository`、`MockTelemetryRepository`、`MockAlertDispatcher` 与 `MockNotificationChannel`。

---

## 2. 手动端到端 (E2E) 验证

### 2.1 MQTT 遥测上报 (`mosquitto_pub`)
由于 `MONITOR_MQTT_STRICT` 默认为 `true`，请先在 `/devices` 将 `pump-01` 注册为 `ACTIVE`（或开发环境下设置 `MONITOR_MQTT_STRICT=false`），再向 `noxfort/telemetry/#` 发布消息：

```bash
# 保活心跳
mosquitto_pub -t "noxfort/telemetry/pump-01" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# 严重告警（通过 telemetry_id 关联至 alert_dispatch_logs）
mosquitto_pub -t "noxfort/telemetry/pump-01" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "CRITICAL",
  "message": "液压管道过压报警: 180 bar",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 Zero-Trust HTTP REST 验证 (`cURL`)

```bash
# 1. 提交设备预配对请求 (202 Accepted，不返回密钥)
curl -i -X POST http://localhost:22100/api/pair \
  -H "Content-Type: application/json" \
  -d '{"origin":"carina-core","name":"Carina Core"}'

# 2. 在 /devices 审批后复制临时令牌 nox_live_ (TTL 5m)，再上报遥测：
curl -i -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_YOUR_COPIED_TOKEN" \
  -H "X-Device-Origin: carina-core" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "模型推理耗时 4.2ms (预期上限 3.5ms)",
    "occurred_at": "'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"
  }'
```
*预期结果*：`202 Accepted`，同时 `GET /api/dashboard/summary` 与 `GET /metrics` 计数器实时刷新。

---

## 3. 可观测性与演示模式

```bash
curl -i http://localhost:22100/healthz
curl -s http://localhost:22100/api/health
curl -s http://localhost:22100/metrics
make record-demo    # 针对 --demo 模式执行 scripts/demo/ 演示流水线
```
