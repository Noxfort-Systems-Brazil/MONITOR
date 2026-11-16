# 👨‍💻 Руководство для разработчиков: Noxfort Monitor™

Настоящий документ содержит стандарты разработки и сопровождения **Noxfort Monitor™ v2.0**: команды `Makefile`, контракт `ReloadableRepository`, HTTP-утилиты безопасности, метрики Prometheus и управление фоновыми процессами.

⬅️ [Центр документации](README.md) | 🏛️ [Архитектура](architecture.md) | 🧪 [Тестирование](testing.md) | 🗄️ [База данных](database.md)

---

## 1. Команды `Makefile`

| Команда | Описание |
| :--- | :--- |
| `make build` | Сборка десктопного бинарного файла (`bin/noxfort-monitor`). |
| `make run` | Запуск в режиме разработки с интерфейсом Wails v2. |
| `make run-headless` | Запуск сервера в фоновом режиме `--headless`. |
| `make test` | Запуск тестов Go (`go test ./... -v`) + фронтенд Vitest (`npm test`). |
| `make test-frontend` | Запуск только JavaScript-тестов Vitest (`npm test`). |
| `make lint` | Запуск статического анализа Go (`go vet`). |
| `make record-demo` | Запуск сценариев `scripts/demo/` для `--demo`. |
| `make deb` / `make install-deb` / `make uninstall-deb` | Управление пакетом `.deb`. |
| `make broker-start` / `make broker-stop` | Управление локальным брокером Mosquitto. |

---

## 2. Паттерны, персистентность и безопасность

* **Корень композиции**: Вся инициализация выполняется явно в [`cmd/server/main.go`](../../cmd/server/main.go) с чтением `.env` через `LoadDotEnv()`.
* **Контракт `ReloadableRepository` (`7` репозиториев)**: `Device`, `Telemetry`, `Settings`, `Contact`, `Audit`, `User` и `DBSessionStore` реализуют `SetDB(db *sql.DB, driver string)` для горячей миграции (`MigrateData` в одной транзакции по 9 таблицам) между SQLite WAL и PostgreSQL.
* **HTTP-утилиты и безопасность**:
  * Отрисовывайте страницы через `RenderPageWithRequest`, а отказы доступа оформляйте через `writeForbiddenResponse` (JSON `403` или [`web/templates/forbidden.html`](../../web/templates/forbidden.html)).
  * Применяйте `ExtractSessionToken` (отклоняет `?token=`), `ExtractClientIP` и `IsHTTPSRequest` (`TRUSTED_PROXIES`, `MONITOR_USE_HTTPS`).
  * Скрывайте `ApiKey` и `duckdns_token` при рендеринге HTML.
* **Метрики, оповещения и завершение (`bgCtx`)**:
  * Обновляйте счетчики `internal/metrics` (`IncTelemetryIngested`, `IncTelemetryRejected`, `IncAlertDispatched`, `SetBufferDepth`).
  * Обычная телеметрия проходит через `BufferedTelemetryWriter`, тогда как переходы состояний сохраняются синхронно через `SaveEventReturningID` + `TriggerAlertWithTelemetryID`; завершайте работу через `AlertService.Shutdown(shutdownCtx)`.
