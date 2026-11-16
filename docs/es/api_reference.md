# 📡 Referencia de API y Protocolos de Ingestión

Este documento especifica formalmente las interfaces de comunicación de **Noxfort Monitor™ v2.0**, abarcando la ingestión asíncrona MQTT, el emparejamiento e ingestión HTTP Zero-Trust, las API de administración con RBAC y la observabilidad Prometheus.

⬅️ [Hub Central](../README.md) | 🏛️ [Arquitectura](architecture.md) | 🗄️ [Base de Datos](database.md) | 🔐 [Seguridad](security.md)

---

## 1. Protocolo de Ingestión MQTT

* **Dirección del Broker**: `tcp://127.0.0.1:1883` (configurable en `/settings` con `Reconnect(brokerURL)` en caliente y reintento cada 5s al arrancar)
* **Patrón de Tópico por Defecto**: `noxfort/telemetry/{identifier}`
* **Suscripción con Comodín**: `noxfort/telemetry/#`
* **Política Estrita de Dispositivos**: Controlada por `MONITOR_MQTT_STRICT` (por defecto `true`). Descarta mensajes de dispositivos no registrados o en estado `PAUSED`.

### Formato JSON Universal `IncomingEvent`

```json
{
  "category": "HARDWARE",
  "origin": "sensor-presion-01",
  "level": "CRITICAL",
  "message": "Sobrecalentamiento detectado: Temperatura 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

#### Especificación de Campos:
* **`category`** (*String*): `HARDWARE` o `SOFTWARE`. Determina el enrutamiento RBAC (Hardware $\rightarrow$ Técnicos; Software $\rightarrow$ Programadores).
* **`origin`** (*String*): Identificador único del dispositivo (ej: `carina`, `synapse`, `pump-01`).
* **`level`** (*String*): `INFO`, `WARNING` o `CRITICAL`.
  * Los mensajes `INFO` con palabras de latido ("*system ok*", "*heartbeat*", "*online*") actualizan `last_seen` sin generar alertas innecesarias.
  * Los mensajes `CRITICAL` disparan persistencia síncrona (`SaveEventReturningID`) y envío multicanal vinculado a `telemetry_id`.
* **`message`** (*String*): Descripción legible del suceso.
* **`occurred_at`** (*String ISO-8601*): Marca temporal UTC (validada en HTTP dentro de una ventana de `±300s`).

---

## 2. Emparejamiento e Ingestión HTTP Zero-Trust

### 2.1 Pre-registro de Dispositivos: `POST /api/pair`
Permite a agentes de campo (**Carina**, **Synapse**, nodos IoT) solicitar alta en cola de aprobación.

* **Puerto**: `22100` (o URL pública HTTPS tras Caddy/Ngrok)
* **Autenticación**: Pública (sujeta a límites: cuerpo máximo `64 KB`, cola máxima de `50` dispositivos `PENDING` $\rightarrow$ `429 Too Many Requests`).
* **Cuerpo JSON**: `{"origin": "synapse-core", "name": "Synapse Core"}`
* **Respuestas**:
  * `202 Accepted`: Registrado en estado `PENDING`. **No devuelve ninguna clave API en la respuesta** (el administrador autoriza y copia la clave `nox_live_` de TTL 5 min fuera de banda desde `/devices`).
  * `409 Conflict`: El dispositivo ya está `ACTIVE`.

### 2.2 Ingestión Telemétrica: `POST /api/telemetry`
* **Autenticación**: Obligatoria mediante `X-Device-Token: <api_key>` o `Authorization: Bearer <api_key>` (comparación en tiempo constante).
* **Anti-Spoofing y Replay**: Valida la cabecera `X-Device-Origin` contra el origen registrado y rechaza `occurred_at` fuera de `±300s` (`±5m`).
* **Respuestas**: `202 Accepted` (encolado en `BufferedTelemetryWriter`), `401 Unauthorized` (token inválido/expirado) o `403 Forbidden` (dispositivo `PAUSED` o dismatch de origen).

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_tu_token" \
  -H "X-Device-Origin: synapse-core" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Pool de conexiones al 85% de capacidad.",
    "occurred_at": "2026-09-27T18:00:00Z"
  }'
```

---

## 3. Endpoints de Administración y Operación

Las rutas web/API protegidas usan cookies HTTP-only (`DBSessionStore`), rechazan `?token=` en URL y aplican RBAC estricto (`OPERATOR` solo lectura + autogestión de contraseña).

| Endpoint | Método | Rol Mínimo | Descripción |
|---|---|---|---|
| `/api/auth/login` | `POST` | Público | Autentica usuario y emite cookie `noxfort_session`; devuelve `429` tras 5 fallos en 15m (bloqueo 5m). |
| `/api/auth/logout` | `POST` | Autenticado | Revoca el hash de sesión en la tabla `sessions`. |
| `/api/auth/status` | `GET` | Público | Retorna estado de autenticación, usuario y rol de la sesión. |
| `/api/auth/register` | `POST` | Admin | Registro administrativo con validación de complejidad de contraseña. |
| `/api/dashboard/summary` | `GET` | Operador+ | Resumen JSON en tiempo real (polling cada 10s) en 5 estados: `ONLINE`, `OFFLINE`, `PENDING`, `PAUSED`, `WAITING`. |
| `/devices` | `GET` | Operador+ | Panel de dispositivos (secretos `ApiKey` activos redactados en HTML). |
| `/devices/create` / `update` / `toggle` | `POST` | Admin | Crea dispositivo (clave única), renombra o alterna Modo Mantenimiento (`PAUSED`). |
| `/devices/rotate-key` / `delete` | `POST` | Admin | Rota la clave API del dispositivo o lo elimina permanentemente. |
| `/contacts` y `/contacts/{create,update,delete}` | `GET`/`POST` | Op+ / Admin | Consulta (`Operator+`) y gestión (`Admin`) del equipo de respuesta. |
| `/api/users` y `/api/users/{create,delete}` | `GET`/`POST` | Admin | Gestión de cuentas (protege superusuario y último admin). |
| `/api/users/password` | `POST` | Operador+ | Cambio de contraseña propia (requiere `current_password`). |
| `/api/users/role` y `/api/users/unlock` | `POST` | Admin | Actualiza rol RBAC o desbloquea cuentas en `LoginAttemptTracker`. |
| `/api/settings/database/status` / `test` / `save` / `provision-user` | `GET`/`POST` | Op+ / Admin | Diagnóstico y cambio dinámico SQLite $\leftrightarrow$ PostgreSQL con `MigrateData` transacional. |
| `/api/settings/database/backups` / `backup` | `GET`/`POST` | Admin | Lista o genera snapshots calientes (`VACUUM INTO` / `pg_dump -F c`). |
| `/api/settings/database/backups/download` / `restore` | `GET`/`POST` | Admin | Descarga segura (`ResolveBackupPath`) y restauración verificada (`PRAGMA quick_check` / `pg_restore`). |
| `/api/tunnel/{status,save,start,stop,disconnect,test}` | `GET`/`POST` | Op+ / Admin | Consulta (`Operator+`) y control (`Admin`) de DuckDNS y Ngrok. |
| `/api/audit/security` / `alerts` / `transitions` | `GET` | Operador+ | Consulta registros de seguridad, alertas y transiciones de disponibilidad. |
| `/settings/test` y `/settings/test-telegram` | `POST` | Admin | Dispara pruebas SMTP (TLS implícito 465 / CRLF sanitizado) o Telegram MarkdownV2. |

---

## 4. Endpoints Públicos de Observabilidad y Diagnóstico

| Endpoint | Método | Descripción | Formato |
|---|---|---|---|
| `/healthz` | `GET` | Probe de liveness (responde 200 `OK`). | `text/plain` |
| `/api/health` | `GET` | Diagnóstico profundo: estado/latencia de BD, conexión MQTT, memoria, goroutines y nodos. | `application/json` |
| `/metrics` | `GET` | Métricas **Prometheus**: `noxfort_uptime_seconds`, `noxfort_goroutines`, `noxfort_memory_heap_alloc_bytes`, `noxfort_memory_heap_sys_bytes`, `noxfort_memory_num_gc`, `noxfort_db_status`, `noxfort_db_latency_ms`, `noxfort_mqtt_connected`, `noxfort_devices_total`, `noxfort_devices_by_status`, `noxfort_telemetry_buffer_depth`, `noxfort_telemetry_ingested_total`, `noxfort_telemetry_rejected_total`, `noxfort_alerts_dispatched_total`. | `text/plain; version=0.0.4` |
