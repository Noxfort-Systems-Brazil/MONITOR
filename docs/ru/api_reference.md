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
| `/api/devices` | `GET` | Оператор+ | Список зарегистрированного оборудования и статус `LastSeen`. |
| `/api/devices/register` | `POST` | Admin | Регистрация или обновление данных узла. |
| `/api/settings/database/test` | `POST` | Admin | Проверка подключения к PostgreSQL перед переключением. |
| `/api/settings/database/migrate`| `POST` | Admin | Миграция таблиц между SQLite и PostgreSQL. |
| `/api/settings/database/save` | `POST` | Admin | Горячее переключение активных репозиториев на целевой движок. |
| `/api/tunnel/start` | `POST` | Admin | Запуск обратного туннеля Ngrok. |
| `/api/tunnel/stop` | `POST` | Admin | Остановка туннеля. |
| `/api/audit/security` | `GET` | Admin | Журналы безопасности и попыток авторизации. |
| `/api/audit/alerts` | `GET` | Admin | Журналы доставки оповещений и соблюдения SLA. |
