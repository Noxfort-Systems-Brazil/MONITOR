# 📡 Referencia de API y Protocolos de Ingestión

Este documento especifica formalmente las interfaces de comunicación de **Noxfort Monitor™ v2.0**, abarcando la ingestión asíncrona MQTT, el endpoint HTTP REST de telemetría y las API de administración.

⬅️ [Hub Central](../README.md) | 🏛️ [Arquitectura](architecture.md) | 🗄️ [Base de Datos](database.md) | 🔐 [Seguridad](security.md)

---

## 1. Protocolo de Ingestión MQTT

* **Dirección del Broker**: `tcp://127.0.0.1:1883`
* **Patrón de Tópico por Defecto**: `noxfort/devices/{identifier}/telemetry`
* **Suscripción con Comodín**: `noxfort/devices/+/telemetry`

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
  * Los mensajes `INFO` con palabras de latido ("*system ok*", "*heartbeat*", "*online*") actualizan la marca temporal sin generar alertas ni escrituras innecesarias en la base de datos.
  * Los mensajes `CRITICAL` disparan el envío inmediato de alertas multicanal.
* **`message`** (*String*): Descripción legible del suceso.
* **`occurred_at`** (*String ISO-8601*): Marca temporal UTC.

---

## 2. Ingestión Telemétrica por HTTP REST

### `POST /api/telemetry`
Diseñado para agentes de campo (**Carina**, **Synapse**, microcontroladores IoT o scripts cURL).

* **Puerto**: `22100` (o URL pública de DuckDNS/Caddy/Ngrok)
* **Autenticación**: Pública (exenta de middleware para nodos autónomos de campo).
* **Aislamiento de Red**: En producción (`ExternalIngestionHandler`), `POST /api/telemetry`, `/healthz`, `/api/health` y `/metrics` son accesibles en el puerto `22100`. Las solicitudes directas de navegador al panel devuelven **403 Forbidden**.
* **Cabecera**: `Content-Type: application/json`
* **Cuerpo**: Idéntico al esquema JSON `IncomingEvent`.

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
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

| Endpoint | Método | Rol Mínimo | Descripción |
|---|---|---|---|
| `/api/auth/login` | `POST` | Público | Autentica credenciales y emite cookie `noxfort_session`. |
| `/api/auth/logout` | `POST` | Autenticado | Invalida el token de sesión activo. |
| `/api/auth/status` | `GET` | Público | Retorna estado de autenticación, usuario y rol de la sesión. |
| `/api/auth/register` | `POST` | Admin | Alias protegido de creación de cuentas con validación de complejidad de contraseña. |
| `/devices` | `GET` | Operador+ | Renderiza la pantalla de gestión con dispositivos registrados y `LastSeen`. |
| `/devices/delete` | `POST` | Admin | Elimina un dispositivo por formulario POST (`id=xyz`) y redirige (`303`). |
| `/contacts` | `GET` | Operador+ | Renderiza el panel de gestión del Equipo de Respuesta. |
| `/contacts/create` | `POST` | Admin | Registra un nuevo contacto de alerta (nombre, email, teléfono, telegram_chat_id, rol). |
| `/contacts/update` | `POST` | Admin | Actualiza los datos y canales de notificación de un contacto existente. |
| `/contacts/delete` | `POST` | Admin | Elimina un contacto del equipo de respuesta. |
| `/api/users` | `GET` | Admin | Retorna listado JSON con todos los operadores registrados. |
| `/api/users/create` | `POST` | Admin | Crea una cuenta de operador o admin (exige contraseña segura de 8+ caracteres). |
| `/api/users/delete` | `POST` | Admin | Elimina una cuenta de operador (el superusuario está protegido). |
| `/api/settings/database/status` | `GET` | Autenticado | Retorna motor activo (Postgres/SQLite), versión y latencia. |
| `/api/settings/database/test` | `POST` | Admin | Comprueba credenciales de PostgreSQL antes del cambio. |
| `/api/settings/database/save` | `POST` | Admin | Recarga repositorios activos con el motor destino (soporta `migrate=true`). |
| `/api/settings/database/provision-user` | `POST` | Admin | Crea usuario dedicado con permisos restringidos en PostgreSQL. |
| `/api/settings/database/backup` | `POST` | Admin | Dispara snapshot consistente en caliente (`VACUUM INTO` / `pg_dump`). |
| `/api/settings/database/backups` | `GET` | Admin | Lista los snapshots de backup almacenados en `backups/`. |
| `/api/tunnel/status` | `GET` | Autenticado | Retorna estado, proveedor (DuckDNS/Ngrok), URLs y IPv6. |
| `/api/tunnel/save` | `POST` | Admin | Guarda parámetros de DuckDNS / Ngrok en la base de datos. |
| `/api/tunnel/start` | `POST` | Admin | Dispara actualización dinámica DuckDNS o abre túnel. |
| `/api/tunnel/stop` | `POST` | Admin | Detiene actualizaciones activas y desactiva inicio automático. |
| `/api/tunnel/disconnect` | `POST` | Admin | Desvincula credenciales guardadas y apaga el servicio. |
| `/api/tunnel/test` | `POST` | Admin | Valida activamente token y resolución DNS de DuckDNS. |
| `/api/audit/security` | `GET` | Admin | Consulta registros de seguridad y accesos administrativos. |
| `/api/audit/alerts` | `GET` | Admin | Consulta registros de despacho y SLA de alertas. |
| `/api/audit/transitions` | `GET` | Admin | Consulta transiciones de estado de nodos (caídas/recuperaciones). |
| `/settings/test` | `POST` | Admin | Envía correo de prueba usando la configuración SMTP activa. |
| `/settings/test-telegram` | `POST` | Admin | Envía alerta de prueba MarkdownV2 a Telegram. |
| `/api/open-external` | `POST` | Autenticado | Abre URL en el navegador predeterminado del sistema operativo. |
| `/api/window/toggle-fullscreen` | `POST` | Público | Alterna ventana Wails entre pantalla completa y normal. |
| `/api/window/exit-fullscreen` | `POST` | Público | Sale de pantalla completa. |

---

## 4. Endpoints Públicos de Observabilidad y Diagnóstico

*Exentos de autenticación para monitoreo con Prometheus y revisiones de contenedores.*

| Endpoint | Método | Descripción | Formato |
|---|---|---|---|
| `/healthz` | `GET` | Probe de liveness (responde 200 `OK`). | `text/plain` |
| `/api/health` | `GET` | Diagnóstico profundo: estado y latencia de BD, conexión MQTT, memoria, goroutines y nodos. | `application/json` |
| `/metrics` | `GET` | Métricas en formato **Prometheus** (`noxfort_uptime_seconds`, `noxfort_goroutines`, etc.). | `text/plain; version=0.0.4` |
