# 👨‍💻 Guía Avanzada para Desarrolladores: Noxfort Monitor™

Este documento proporciona pautas de ingeniería para compilar, mantener y extender **Noxfort Monitor™ v2.0**, cubriendo el `Makefile`, arquitectura SOLID, repositorios con hot-reload (`DBManager`), auxiliares HTTP de seguridad, métricas Prometheus y apagado limpio.

⬅️ [Centro de Documentación](README.md) | 🏛️ [Arquitectura](architecture.md) | 🧪 [Pruebas](testing.md) | 🗄️ [Base de Datos](database.md)

---

## 1. Entorno Local y Comandos de `Makefile`

| Comando | Descripción |
| :--- | :--- |
| `make build` | Compila el binario de escritorio (`bin/noxfort-monitor`). |
| `make run` | Ejecuta la aplicación completa con interfaz Wails v2. |
| `make run-headless` | Inicia el servidor en modo `--headless`. |
| `make test` | Ejecuta pruebas Go (`go test ./... -v`) + frontend Vitest (`npm test`). |
| `make test-frontend` | Ejecuta únicamente la suite JavaScript con Vitest. |
| `make lint` | Ejecuta verificación estática (`go vet`). |
| `make record-demo` | Ejecuta scripts automatizados en `scripts/demo/` contra `--demo`. |
| `make deb` / `make install-deb` / `make uninstall-deb` | Empaqueta e instala/desinstala el paquete Debian `.deb`. |
| `make broker-start` / `make broker-stop` | Controla el broker Mosquitto local. |

---

## 2. Patrones de Diseño, Persistencia y Seguridad

* **Raíz de Composición**: Instanciación explícita en [`cmd/server/main.go`](../../cmd/server/main.go) cargando `.env` mediante `LoadDotEnv()`.
* **Contrato `ReloadableRepository` (7 Repositorios)**: Todos los repositorios (`Device`, `Telemetry`, `Settings`, `Contact`, `Audit`, `User` y `DBSessionStore`) implementan `SetDB(db *sql.DB, driver string)` para permitir cambios SQLite WAL $\leftrightarrow$ PostgreSQL con migración transacional (`MigrateData`) sin reiniciar.
* **Auxiliares HTTP y Seguridad**:
  * Renderice vistas mediante `RenderPageWithRequest` y rechace permisos mediante `writeForbiddenResponse` (JSON `403` en APIs o [`web/templates/forbidden.html`](../../web/templates/forbidden.html) en navegador/Wails).
  * Utilice siempre `ExtractSessionToken` (prohíbe `?token=`), `ExtractClientIP` e `IsHTTPSRequest` (validando `TRUSTED_PROXIES` y `MONITOR_USE_HTTPS`).
  * Mantenga redactados `ApiKey` y `duckdns_token` en salidas HTML.
* **Métricas, Alertas y Ciclo de Vida (`bgCtx`)**:
  * Instrumente contadores con `internal/metrics` (`IncTelemetryIngested`, `IncTelemetryRejected`, `IncAlertDispatched`, `SetBufferDepth`).
  * Use `BufferedTelemetryWriter` para ingesta general y `SaveEventReturningID` + `TriggerAlertWithTelemetryID` en transiciones críticas; drene pendientes en cierre mediante `AlertService.Shutdown(shutdownCtx)`.
