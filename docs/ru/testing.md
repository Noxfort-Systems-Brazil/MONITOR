# 🧪 Тестирование и контроль качества (QA)

Документ описывает процессы проверки **Noxfort Monitor™ v2.0**: автоматические тесты (Go + Vitest), ручную проверку E2E по MQTT (`noxfort/telemetry/#`) и Zero-Trust HTTP (`/api/pair` + `/api/telemetry`), а также режим демонстрации.

⬅️ [Главный Хаб](../README.md) | 🏛️ [Архитектура](architecture.md) | 📡 [API](api_reference.md)

---

## 1. Автоматизированные наборы тестов

```bash
# Запуск backend-тестов Go (go test ./... -v) и frontend-тестов Vitest (npm test)
make test

# Только фронтенд (Vitest: 4 тестовых файла, 26 тестов; заменил прежний pytest)
make test-frontend

# Статический анализ Go (go vet)
make lint
```

* **Backend Go**: Покрывает безопасность (`lockout_test.go`, PBKDF2, сложность паролей), хранилище (`session_repo_test.go`, `retention_test.go`, миграцию, бэкапы/восстановление), HTTP RBAC/прием данных, строгий MQTT, декодер и оповещения (`alerts_test.go`).
* **Моки в памяти**: `MockDeviceRepository`, `MockTelemetryRepository`, `MockAlertDispatcher` и `MockNotificationChannel`.

---

## 2. Ручное E2E-тестирование

### 2.1 Прием через MQTT (`mosquitto_pub`)
При `MONITOR_MQTT_STRICT=true` (по умолчанию) предварительно создайте узел `pump-01` со статусом `ACTIVE` в `/devices` (или задайте `MONITOR_MQTT_STRICT=false` локально) перед публикацией в `noxfort/telemetry/#`:

```bash
# Проверка пульса (Keep-Alive)
mosquitto_pub -t "noxfort/telemetry/pump-01" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# Критическая тревога (связывается в alert_dispatch_logs через telemetry_id)
mosquitto_pub -t "noxfort/telemetry/pump-01" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "CRITICAL",
  "message": "Авария гидравлики: Давление 180 бар",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 Прием Zero-Trust через HTTP REST (`cURL`)

```bash
# 1. Предварительная регистрация (202 Accepted без выдачи ключа)
curl -i -X POST http://localhost:22100/api/pair \
  -H "Content-Type: application/json" \
  -d '{"origin":"carina-core","name":"Carina Core"}'

# 2. Подтвердите узел в /devices, скопируйте токен nox_live_ (TTL 5м) и отправьте событие:
curl -i -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_VASH_TOKEN" \
  -H "X-Device-Origin: carina-core" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "Задержка вывода нейросети 4.2ms (норма: 3.5ms)",
    "occurred_at": "'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"
  }'
```
*Ожидаемый результат*: `202 Accepted`, обновление `GET /api/dashboard/summary` и счетчиков на `GET /metrics`.

---

## 3. Автоматизированный режим Demo
Запустите `make record-demo` (`scripts/demo/record_presentation.cjs` и `mock_telemetry.cjs`) против `--demo` (`RunDesktopServer()`).
