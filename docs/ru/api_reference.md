# 📡 Справочник API и протоколов приема телеметрии

Данный документ описывает интерфейсы **Noxfort Monitor™ v2.0**: асинхронный MQTT, Zero-Trust сопряжение и прием телеметрии по HTTP, административное REST API с RBAC и экспорт метрик Prometheus.

⬅️ [Главный Хаб](../README.md) | 🏛️ [Архитектура](architecture.md) | 🗄️ [База данных](database.md) | 🔐 [Безопасность](security.md)

---

## 1. Протокол телеметрии MQTT

* **Адрес брокера**: `tcp://127.0.0.1:1883` (настраивается в `/settings`, поддерживает горячий `Reconnect(brokerURL)` и повторные попытки каждые 5с при запуске)
* **Шаблон топика публикации**: `noxfort/telemetry/{identifier}`
* **Подписка Wildcard**: `noxfort/telemetry/#`
* **Строгий режим (`MONITOR_MQTT_STRICT`)**: По умолчанию `true`; отбрасывает сообщения от незарегистрированных или поставленных на паузу (`PAUSED`) устройств.

### Универсальная схема JSON `IncomingEvent`

```json
{
  "category": "HARDWARE",
  "origin": "sensor-presion-01",
  "level": "CRITICAL",
  "message": "Обнаружен перегрев: температура 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

* **`category`**: `HARDWARE` или `SOFTWARE` (маршрутизация тревог по ролям контактов).
* **`origin`**: Уникальный идентификатор устройства (`carina`, `synapse`, `pump-01`).
* **`level`**: `INFO`, `WARNING` или `CRITICAL` (`INFO`-пульсы обновляют `last_seen` без лишних тревог; `CRITICAL` запускает синхронный `SaveEventReturningID` и оповещения со ссылкой на `telemetry_id`).
* **`message`**: Читаемое описание события.
* **`occurred_at`**: Временная метка UTC ISO-8601 (в HTTP проверяется окно `±300s`).

---

## 2. Zero-Trust сопряжение и прием по HTTP

### 2.1 Предварительная регистрация: `POST /api/pair`
* **Порт**: `22100` (или публичный HTTPS-адрес за Caddy/Ngrok)
* **Доступ**: Публичный (тело до `64 КБ`, лимит очереди `50` записей `PENDING` $\rightarrow$ `429 Too Many Requests`).
* **JSON-запрос**: `{"origin": "synapse-core", "name": "Synapse Core"}`
* **Ответы**:
  * `202 Accepted`: Устройство добавлено в очередь `PENDING`. **API-ключ в ответе не возвращается** (`ADMIN` подтверждает узел и копирует временный токен `nox_live_` со сроком действия 5 мин в `/devices`).
  * `409 Conflict`: Устройство уже зарегистрировано и имеет статус `ACTIVE`.

### 2.2 Прием телеметрии: `POST /api/telemetry`
* **Аутентификация**: Обязательный заголовок `X-Device-Token: <api_key>` или `Authorization: Bearer <api_key>` (сравнение за константное время).
* **Anti-Spoofing и Replay**: Заголовок `X-Device-Origin` сверяется с источником, а метка `occurred_at` должна укладываться в допуск `±300s`.
* **Ответы**: `202 Accepted` (`BufferedTelemetryWriter`), `401 Unauthorized` или `403 Forbidden` (устройство в `PAUSED` или несоответствие `origin`).

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_your_token" \
  -H "X-Device-Origin: synapse-core" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Пул соединений заполнен на 85%.",
    "occurred_at": "2026-09-27T18:00:00Z"
  }'
```

---

## 3. Эндпоинты администрирования и эксплуатации

Защищенные маршруты используют cookie-сессии (`DBSessionStore`), отклоняют передачу токена через `?token=` и соблюдают строгий RBAC (`OPERATOR` только чтение + смена собственного пароля).

| Эндпоинт | Метод | Мин. роль | Описание |
|---|---|---|---|
| `/api/auth/login` | `POST` | Публичный | Аутентификация и cookie `noxfort_session`; `429` после 5 ошибок за 15м (блокировка 5м). |
| `/api/auth/logout` | `POST` | Авторизован | Удаляет хеш текущей сессии из таблицы `sessions`. |
| `/api/auth/status` | `GET` | Публичный | Текущий статус сессии, имя пользователя и роль. |
| `/api/auth/register` | `POST` | Admin | Регистрация учетной записи с проверкой сложности пароля. |
| `/api/dashboard/summary` | `GET` | Оператор+ | Живая сводка JSON (опрос каждые 10с) по 5 состояниям: `ONLINE`, `OFFLINE`, `PENDING`, `PAUSED`, `WAITING`. |
| `/devices` | `GET` | Оператор+ | Список устройств (активные секреты `ApiKey` скрыты в HTML). |
| `/devices/create` / `update` / `toggle` | `POST` | Admin | Создание (одноразовый показ ключа), переименование или переключение режима обслуживания (`PAUSED`). |
| `/devices/rotate-key` / `delete` | `POST` | Admin | Ротация API-ключа устройства или его удаление. |
| `/contacts` и `/contacts/{create,update,delete}` | `GET`/`POST` | Опер.+ / Admin | Просмотр (`Operator+`) и управление (`Admin`) контактами оповещений. |
| `/api/users` и `/api/users/{create,delete}` | `GET`/`POST` | Admin | Управление пользователями (защита суперпользователя и последнего админа). |
| `/api/users/password` | `POST` | Оператор+ | Самостоятельная смена своего пароля (требует `current_password`). |
| `/api/users/role` и `/api/users/unlock` | `POST` | Admin | Изменение роли RBAC или разблокировка учетной записи в `LoginAttemptTracker`. |
| `/api/settings/database/status` / `test` / `save` / `provision-user` | `GET`/`POST` | Опер.+ / Admin | Диагностика и переключение SQLite $\leftrightarrow$ PostgreSQL через транзакционный `MigrateData`. |
| `/api/settings/database/backups` / `backup` | `GET`/`POST` | Admin | Просмотр списка и создание горячих резервных копий (`VACUUM INTO` / `pg_dump -F c`). |
| `/api/settings/database/backups/download` / `restore` | `GET`/`POST` | Admin | Защищенное скачивание (`ResolveBackupPath`) и восстановление (`PRAGMA quick_check` / `pg_restore`). |
| `/api/tunnel/{status,save,start,stop,disconnect,test}` | `GET`/`POST` | Опер.+ / Admin | Просмотр (`Operator+`) и управление (`Admin`) DuckDNS и Ngrok. |
| `/api/audit/security` / `alerts` / `transitions` | `GET` | Оператор+ | Журналы безопасности, доставки оповещений и переходов состояний. |
| `/settings/test` и `/settings/test-telegram` | `POST` | Admin | Тестовая отправка SMTP (TLS 465 / фильтрация CRLF) и Telegram MarkdownV2. |

---

## 4. Публичные эндпоинты наблюдаемости

| Эндпоинт | Метод | Описание | Формат |
|---|---|---|---|
| `/healthz` | `GET` | Liveness-проба (`200 OK`). | `text/plain` |
| `/api/health` | `GET` | Глубокая диагностика: статус/задержка БД, MQTT, память, горутины и узлы. | `application/json` |
| `/metrics` | `GET` | Метрики **Prometheus**: `noxfort_uptime_seconds`, `noxfort_goroutines`, `noxfort_memory_heap_alloc_bytes`, `noxfort_memory_heap_sys_bytes`, `noxfort_memory_num_gc`, `noxfort_db_status`, `noxfort_db_latency_ms`, `noxfort_mqtt_connected`, `noxfort_devices_total`, `noxfort_devices_by_status`, `noxfort_telemetry_buffer_depth`, `noxfort_telemetry_ingested_total`, `noxfort_telemetry_rejected_total`, `noxfort_alerts_dispatched_total`. | `text/plain; version=0.0.4` |
