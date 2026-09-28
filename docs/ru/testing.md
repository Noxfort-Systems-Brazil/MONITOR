# 🧪 Тестирование и контроль качества (QA)

Данный документ описывает процессы тестирования **Noxfort Monitor™ v2.0**, включая автоматизированные модульные тесты с моками и ручную проверку сквозного взаимодействия (E2E) через MQTT и HTTP REST.

⬅️ [Главный Хаб](../README.md) | 🏛️ [Архитектура](architecture.md) | 📡 [API](api_reference.md)

---

## 1. Автоматизированное тестирование

Запуск полного набора тестов:
```bash
make test
# Эквивалентно: go test ./... -v
```

### Архитектура моков в оперативной памяти
Бизнес-логика в пакетах `internal/monitor` и `internal/security` полностью изолирована от дисковых баз данных с помощью мок-интерфейсов:
* `MockDeviceRepository`: Имитирует чтение устройств и обновление времени `LastSeen`.
* `MockTelemetryRepository`: Проверяет корректность сохранения инцидентов.
* `MockAlertDispatcher`: Проверяет передачу событий в модуль маршрутизации.
* `MockNotificationChannel`: Тестирует отправку без реальных сетевых запросов к серверам SMTP и Telegram.

---

## 2. Ручное E2E-тестирование

### 2.1 Прием через MQTT (`mosquitto_pub`)

```bash
# Проверка пульса (Keep-Alive)
mosquitto_pub -t "noxfort/devices/pump-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# Критическая тревога (Инициирует отправку уведомлений)
mosquitto_pub -t "noxfort/devices/pump-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "CRITICAL",
  "message": "Авария гидравлики: Давление 180 бар",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 Прием через HTTP REST (`cURL`)

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "Задержка вывода нейросети 4.2ms (норма: 3.5ms)",
    "occurred_at": "2026-09-27T18:10:00Z"
  }'
```
*Ожидаемый ответ*: HTTP `200 OK` с телом `{"status":"received"}`.
