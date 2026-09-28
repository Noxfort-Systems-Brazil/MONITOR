# 🧪 Pruebas y Aseguramiento de Calidad (QA)

Este documento especifica los flujos de validación de **Noxfort Monitor™ v2.0**, abarcando pruebas unitarias automáticas con mocks y verificación manual E2E mediante MQTT y HTTP REST.

⬅️ [Hub Central](../README.md) | 🏛️ [Arquitectura](architecture.md) | 📡 [API](api_reference.md)

---

## 1. Suite de Pruebas Automatizadas

Ejecución de la suite completa de pruebas:
```bash
make test
# Equivalente a: go test ./... -v
```

### Arquitectura de Mocks en Memoria
La lógica de negocio en `internal/monitor` e `internal/security` no requiere bases de datos en disco gracias a interfaces simuladas:
* `MockDeviceRepository`: Simula la consulta y actualización de `LastSeen`.
* `MockTelemetryRepository`: Verifica la persistencia de incidentes.
* `MockAlertDispatcher`: Valida que las alertas lleguen al enrutador.
* `MockNotificationChannel`: Comprueba el envío sin realizar llamadas reales a SMTP o Telegram.

---

## 2. Verificación Manual de Extremo a Extremo (E2E)

### 2.1 Ingestión mediante MQTT (`mosquitto_pub`)

```bash
# Latido de Presencia (Keep-Alive)
mosquitto_pub -t "noxfort/devices/bomba-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "bomba-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# Alarma Crítica (Dispara Alertas Inmediatas)
mosquitto_pub -t "noxfort/devices/bomba-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "bomba-01",
  "level": "CRITICAL",
  "message": "Alarma de Sobrepresión Hidráulica: 180 bar",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 Ingestión mediante HTTP REST (`cURL`)

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "Latencia de inferencia 4.2ms (objetivo: 3.5ms)",
    "occurred_at": "2026-09-27T18:10:00Z"
  }'
```
*Respuesta esperada*: Código HTTP `200 OK` con cuerpo `{"status":"received"}`.
