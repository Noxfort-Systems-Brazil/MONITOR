# 🧪 Pruebas y Aseguramiento de Calidad (QA)

Este documento especifica los flujos de validación de **Noxfort Monitor™ v2.0**, abarcando las suites automatizadas (Go + Vitest), verificación E2E en MQTT (`noxfort/telemetry/#`) y HTTP Zero-Trust (`/api/pair` + `/api/telemetry`), y el modo de demostración automatizada.

⬅️ [Hub Central](../README.md) | 🏛️ [Arquitectura](architecture.md) | 📡 [API](api_reference.md)

---

## 1. Suites de Pruebas Automatizadas

```bash
# Ejecuta backend Go (go test ./... -v) + frontend Vitest (npm test)
make test

# Solo pruebas de frontend (Vitest: 4 archivos, 26 pruebas; reemplaza antiguo pytest)
make test-frontend

# Análisis estático (go vet)
make lint
```

* **Backend Go**: Cubre seguridad (`lockout_test.go`, hashing PBKDF2, complejidad), almacenamiento (`session_repo_test.go`, `retention_test.go`, migración transacional, backup/restore), HTTP RBAC/ingestión, MQTT estricto, decodificador de protocolo y alertas (`alerts_test.go`).
* **Mocks en Memoria**: `MockDeviceRepository`, `MockTelemetryRepository`, `MockAlertDispatcher` y `MockNotificationChannel` permiten validar reglas sin dependencias externas.

---

## 2. Verificación Manual de Extremo a Extremo (E2E)

### 2.1 Ingestión mediante MQTT (`mosquitto_pub`)
Con `MONITOR_MQTT_STRICT=true` (por defecto), registre primero `bomba-01` como `ACTIVE` en `/devices` (o ajuste `MONITOR_MQTT_STRICT=false` en entorno de pruebas local) antes de publicar bajo `noxfort/telemetry/#`:

```bash
# Latido de Presencia (Keep-Alive)
mosquitto_pub -t "noxfort/telemetry/bomba-01" -m '{
  "category": "HARDWARE",
  "origin": "bomba-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# Alarma Crítica (Dispara Alertas y Registro vinculado en alert_dispatch_logs)
mosquitto_pub -t "noxfort/telemetry/bomba-01" -m '{
  "category": "HARDWARE",
  "origin": "bomba-01",
  "level": "CRITICAL",
  "message": "Alarma de Sobrepresión Hidráulica: 180 bar",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 Ingestión Zero-Trust mediante HTTP REST (`cURL`)

```bash
# 1. Pre-registro Zero-Trust (responde 202 Accepted sin devolver clave)
curl -i -X POST http://localhost:22100/api/pair \
  -H "Content-Type: application/json" \
  -d '{"origin":"carina-core","name":"Carina Core"}'

# 2. Aprobar en /devices y copiar el token temporal nox_live_ (TTL 5m), luego enviar telemetría:
curl -i -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_TU_TOKEN_COPIADO" \
  -H "X-Device-Origin: carina-core" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "Latencia de inferencia 4.2ms (objetivo: 3.5ms)",
    "occurred_at": "'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"
  }'
```
*Respuesta esperada*: `202 Accepted`, actualización inmediata en `GET /api/dashboard/summary` y contadores en `GET /metrics`.

---

## 3. Modo Demo Automatizado
Ejecute `make record-demo` (`scripts/demo/record_presentation.cjs` y `mock_telemetry.cjs`) contra `--demo` (`RunDesktopServer()`) para validar flujos E2E en navegador.
