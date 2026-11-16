# 🔬 Журнал технических решений и исследований: Noxfort Monitor™

Документ обобщает архитектурные решения (ADR) и техническую эволюцию **Noxfort Monitor™ v2.0**.

⬅️ [Центр документации](README.md) | 🏛️ [Архитектура](architecture.md) | 🗄️ [База данных](database.md) | 🌐 [Удаленный доступ](remote_access.md)

---

## 1. Удаление `configs/config.yaml` в пользу `.env` + SQL `settings`
* **Решение**: Файл `configs/config.yaml` удален во избежание дублирования конфигурации. Параметры запуска загружаются из `.env` (`LoadDotEnv()`), а динамические настройки (URL MQTT, оповещения, туннели) хранятся в SQL `settings` с переподключением на лету (`SetOnMQTTConfigChanged` $\rightarrow$ `Reconnect`).

## 2. Zero-Trust сопряжение (`POST /api/pair`) и политика `MONITOR_MQTT_STRICT`
* **Решение**: `POST /api/pair` добавляет устройства в очередь `PENDING` (макс. 50, `409 Conflict` если уже `ACTIVE`) с ответом `202` **без выдачи ключа**; токен `nox_live_` (TTL 5м) копируется вручную в `/devices` и проверяется в `POST /api/telemetry` с `X-Device-Origin` и окном `±300s`. Для `noxfort/telemetry/#` политика `MONITOR_MQTT_STRICT=true` отклоняет незарегистрированные и `PAUSED` узлы.

## 3. Постоянные хешированные сессии, PBKDF2 lockout и строгий RBAC
* **Решение**: Отказ от сессий в памяти в пользу `DBSessionStore` (`SHA-256` в таблице `sessions`, TTL 24ч, `ReloadableRepository`), блокировка `?token=`, `PBKDF2Hasher` (`100 000` итераций) + `LoginAttemptTracker` (5/15м/5м $\rightarrow$ `429`), а также перевод `OPERATOR` в режим строго только для чтения (+ смена собственного пароля).

## 4. Двухдвижковое хранение: SQLite WAL, атомарная миграция, бэкапы и ретеншн
* **Решение**: Чистый Go SQLite (`modernc.org/sqlite`) в режиме `WAL`; `MigrateData` в одной транзакции по 9 таблицам; `BackupManager` (`VACUUM INTO`, `pg_dump -F c`, скачивание + `PRAGMA quick_check`/`pg_restore`) и `RetentionManager` (`MONITOR_RETENTION_DAYS=90`).

## 5. Prometheus-метрики, связка тревог и режим `--demo`
* **Решение**: Добавлен пакет `internal/metrics` (`/metrics`), прямая привязка `alert_dispatch_logs.telemetry_id`, ретраи с бэкоффом (`500ms`/`1500ms`) + `AlertService.Shutdown(ctx)` и флаг `--demo`.
