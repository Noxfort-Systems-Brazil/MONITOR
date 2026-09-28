# 🧪 Testing & Quality Assurance (QA)

This document specifies testing workflows for **Noxfort Monitor™ v2.0**, covering automated unit tests, in-memory repository mocks, and manual E2E validation via MQTT and HTTP REST.

⬅️ [Central Hub](../README.md) | 🏛️ [Architecture](architecture.md) | 📡 [API](api_reference.md)

---

## 1. Automated Test Suite

Run the full automated test suite:
```bash
make test
# Equivalent to: go test ./... -v
```

### In-Memory Mocks Architecture
Core business logic in `internal/monitor` and `internal/security` decouples from disk databases using mock interfaces:
* `MockDeviceRepository`: Simulates node lookup and `LastSeen` updates.
* `MockTelemetryRepository`: Verifies incident persistence.
* `MockAlertDispatcher`: Validates that alerts reach the routing policy.
* `MockNotificationChannel`: Tests channel delivery without actual SMTP or Telegram network calls.

---

## 2. Manual E2E Ingestion Verification

### 2.1 MQTT Ingestion (`mosquitto_pub`)

```bash
# Heartbeat (Keep-Alive)
mosquitto_pub -t "noxfort/devices/pump-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# Critical Alarm (Triggers Alert Dispatch)
mosquitto_pub -t "noxfort/devices/pump-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "CRITICAL",
  "message": "Hydraulic Overpressure Alarm: 180 bar",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 HTTP REST Ingestion (`cURL`)

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "Model inference latency 4.2ms (exceeded 3.5ms target)",
    "occurred_at": "2026-09-27T18:10:00Z"
  }'
```
*Expected response*: HTTP `200 OK` with body `{"status":"received"}`.
