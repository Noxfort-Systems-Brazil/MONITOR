# 🔬 Notas de Investigación y Decisiones Técnicas: Noxfort Monitor™

Este documento consolida el registro de decisiones arquitectónicas (ADRs) y la evolución técnica de **Noxfort Monitor™ v2.0**.

⬅️ [Centro de Documentación](README.md) | 🏛️ [Arquitectura](architecture.md) | 🗄️ [Base de Datos](database.md) | 🌐 [Acceso Remoto](remote_access.md)

---

## 1. Eliminación de `configs/config.yaml` en Favor de `.env` + SQL `settings`
* **Decisión**: Se retiró `configs/config.yaml` para eliminar configuración duplicada. Parámetros de entorno cargan con `LoadDotEnv()` desde `.env`, mientras ajustes operativos (URL MQTT, alertas, túneis) residen en SQL `settings` con hooks en caliente (`SetOnMQTTConfigChanged` $\rightarrow$ `Reconnect`).

## 2. Pareamento Zero-Trust (`POST /api/pair`) y Política `MONITOR_MQTT_STRICT`
* **Decisión**: `POST /api/pair` admite alta previa como `PENDING` (máximo 50, `409 Conflict` si ya `ACTIVE`) respondiendo `202` **sin claves**; las credenciales `nox_live_` (TTL 5m) se aprueban fuera de banda en `/devices` y se validan en `POST /api/telemetry` junto a `X-Device-Origin` y ventana `±300s`. En MQTT, `MONITOR_MQTT_STRICT=true` descarta dispositivos desconocidos o `PAUSED` en `noxfort/telemetry/#`.

## 3. Sesiones Persistentes Hasheadas, Lockout PBKDF2 y RBAC Estricto
* **Decisión**: Sustituido el mapa de sesiones en memoria por `DBSessionStore` (hashes `SHA-256` en tabla `sessions`, TTL 24h, miembro de `ReloadableRepository`), prohibido `?token=`, añadido `PBKDF2Hasher` (`100.000` iteraciones) + `LoginAttemptTracker` (5/15m/5m bloqueo `429`), y restringido `OPERATOR` a modo estrictamente de solo lectura (+ autogestión de contraseña).

## 4. Persistencia Dual-Engine: SQLite WAL, Migración Atómica, Backups y Retención
* **Decisión**: SQLite puro en Go (`modernc.org/sqlite`) configurado con `WAL`; `MigrateData` transacional único sobre 9 tablas aborta `DBManager.Switch` ante fallos; añadidos `BackupManager` (`VACUUM INTO`, `pg_dump -F c`, descarga + `PRAGMA quick_check` / `pg_restore`) y `RetentionManager` (`MONITOR_RETENTION_DAYS=90`).

## 5. Observabilidad Prometheus, Despacho de Alertas y Wails `--demo`
* **Decisión**: Introducido `internal/metrics` (`/metrics`), enlace directo `alert_dispatch_logs.telemetry_id` vía `SaveEventReturningID`, reintentos (`500ms`/`1500ms`) + `AlertService.Shutdown(ctx)` y bandera `--demo` sobre WebKitGTK/HTTP.
