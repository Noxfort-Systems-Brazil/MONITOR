# 📡 Справочник API и протоколы телеметрии

Данный документ описывает сетевые интерфейсы **Noxfort Monitor™ v2.0**, включая асинхронный протокол MQTT, конечную точку HTTP REST и административные API.

⬅️ [Главный Хаб](../README.md) | 🏛️ [Архитектура](architecture.md) | 🗄️ [База данных](database.md) | 🔐 [Безопасность](security.md)

---

## 1. Протокол приема MQTT

* **Адрес брокера**: `tcp://127.0.0.1:1883`
* **Шаблон топика по умолчанию**: `noxfort/devices/{identifier}/telemetry`
* **Маска подписки**: `noxfort/devices/+/telemetry`

### Универсальный формат JSON `IncomingEvent`

```json
{
  "category": "HARDWARE",
  "origin": "sensor-temp-01",
  "level": "CRITICAL",
  "message": "Обнаружен перегрев: Температура 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

#### Описание полей:
* **`category`** (*String*): `HARDWARE` или `SOFTWARE`. Определяет правила RBAC-маршрутизации (Оборудование $\rightarrow$ Техники; ПО $\rightarrow$ Программисты).
* **`origin`** (*String*): Уникальный идентификатор узла (например, `carina`, `synapse`, `pump-01`).
* **`level`** (*String*): `INFO`, `WARNING` или `CRITICAL`.
  * Сообщения `INFO` с ключевыми словами жизнедеятельности ("*system ok*", "*heartbeat*", "*online*") обновляют метку времени без спама оповещений и лишних записей в БД.
  * Сообщения `CRITICAL` инициируют мгновенную рассылку оповещений.
* **`message`** (*String*): Текстовое описание события.
* **`occurred_at`** (*String ISO-8601*): Метка времени в формате UTC.

---

## 2. Прием телеметрии через HTTP REST

### `POST /api/telemetry`
Разработан для автономных полевых агентов (**Carina**, **Synapse**, микроконтроллеров IoT или скриптов cURL).

* **Порт**: `22100` (или публичный URL Ngrok)
* **Аутентификация**: Открытая (без middleware для удобства автономных датчиков).
* **Заголовок**: `Content-Type: application/json`
* **Тело**: Идентично схеме `IncomingEvent`.

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Пул соединений заполнен на 85%.",
    "occurred_at": "2026-09-27T18:00:00Z"
  }'
```

---

## 3. Административные и служебные эндпоинты

| Эндпоинт | Метод | Роль | Назначение |
|---|---|---|---|
| `/api/auth/login` | `POST` | Публичный | Проверка учетных данных и выдача cookie `noxfort_session`. |
| `/api/auth/logout` | `POST` | Авториз. | Аннулирование активного токена сессии. |
| `/api/auth/status` | `GET` | Публичный | Возвращает статус сессии, имя пользователя и роль. |
| `/api/auth/register` | `POST` | Admin | Защищенный эндпоинт регистрации с проверкой сложности пароля. |
| `/devices` | `GET` | Оператор+ | Панель управления зарегистрированным оборудованием и `LastSeen`. |
| `/devices/delete` | `GET` | Admin | Удаление устройства через query-параметр (`?id=xyz`) с перенаправлением (`303`). |
| `/contacts` | `GET` | Оператор+ | Панель управления контактами группы реагирования. |
| `/contacts/create` | `POST` | Admin | Регистрация получателя оповещений (имя, email, телефон, telegram_chat_id, роль). |
| `/contacts/update` | `POST` | Admin | Обновление контактов и каналов оповещения. |
| `/contacts/delete` | `POST` | Admin | Удаление контакта из группы реагирования. |
| `/api/users` | `GET` | Admin | Возвращает JSON-список всех операторов. |
| `/api/users/create` | `POST` | Admin | Создание учетной записи (требует надежный пароль от 8 символов). |
| `/api/users/delete` | `POST` | Admin | Удаление учетной записи оператора (суперпользователь защищен). |
| `/api/settings/database/status` | `GET` | Авториз. | Возвращает активный движок БД (Postgres/SQLite), версию и задержку. |
| `/api/settings/database/test` | `POST` | Admin | Проверка подключения к PostgreSQL перед переключением. |
| `/api/settings/database/save` | `POST` | Admin | Горячее переключение репозиториев на целевой движок (с поддержкой `migrate=true`). |
| `/api/settings/database/provision-user` | `POST` | Admin | Создание выделенного пользователя со строгими правами в схеме PostgreSQL. |
| `/api/settings/database/backup` | `POST` | Admin | Создание согласованного горячего снапшота БД (`VACUUM INTO` / `pg_dump`). |
| `/api/settings/database/backups` | `GET` | Admin | Список всех доступных снапшотов резервных копий в `backups/`. |
| `/api/tunnel/status` | `GET` | Авториз. | Возвращает статус WAN, провайдера (DuckDNS/Ngrok), публичные URL и IPv6. |
| `/api/tunnel/save` | `POST` | Admin | Сохранение настроек DuckDNS / Ngrok в базе данных. |
| `/api/tunnel/start` | `POST` | Admin | Динамическое обновление DNS DuckDNS или запуск туннеля. |
| `/api/tunnel/stop` | `POST` | Admin | Остановка службы и отключение автозапуска при загрузке. |
| `/api/tunnel/disconnect` | `POST` | Admin | Отключение учетных данных из БД и завершение работы службы. |
| `/api/tunnel/test` | `POST` | Admin | Активная проверка токена DuckDNS и разрешения доменных имен. |
| `/api/audit/security` | `GET` | Admin | Журналы безопасности и попыток авторизации. |
| `/api/audit/alerts` | `GET` | Admin | Журналы доставки оповещений и соблюдения SLA. |
| `/api/audit/transitions` | `GET` | Admin | История отказов оборудования и длительности простоев. |
| `/settings/test` | `POST` | Admin | Отправка тестового email через активную конфигурацию SMTP. |
| `/settings/test-telegram` | `POST` | Admin | Отправка тестового оповещения MarkdownV2 в Telegram. |
| `/api/open-external` | `POST` | Авториз. | Открытие URL в браузере по умолчанию операционной системы. |
| `/api/window/toggle-fullscreen` | `POST` | Публичный | Переключение полноэкранного режима окна Wails. |
| `/api/window/exit-fullscreen` | `POST` | Публичный | Выход из полноэкранного режима. |

---

## 4. Публичные эндпоинты мониторинга и диагностики

*Доступны без авторизации для мониторинга Prometheus и проб жизнеспособности контейнеров.*

| Эндпоинт | Метод | Описание | Формат |
|---|---|---|---|
| `/healthz` | `GET` | Проба liveness (возвращает 200 `OK`). | `text/plain` |
| `/api/health` | `GET` | Глубокая диагностика компонентов: БД, MQTT, RAM, горутины и количество узлов. | `application/json` |
| `/metrics` | `GET` | Экспорт метрик в формате **Prometheus** (`noxfort_uptime_seconds`, `noxfort_goroutines` и др.). | `text/plain; version=0.0.4` |
