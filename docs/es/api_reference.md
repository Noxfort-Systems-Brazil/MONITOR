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

* **Puerto**: `22100` (o URL pública de Ngrok)
* **Autenticación**: Pública (exenta de middleware para nodos autónomos de campo).
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
| `/api/devices` | `GET` | Operador+ | Retorna dispositivos registrados y estado `LastSeen`. |
| `/api/devices/register` | `POST` | Admin | Registra o actualiza un nodo de dispositivo. |
| `/api/settings/database/test` | `POST` | Admin | Comprueba credenciales de PostgreSQL antes del cambio. |
| `/api/settings/database/migrate`| `POST` | Admin | Migra esquemas y tablas entre SQLite y PostgreSQL. |
| `/api/settings/database/save` | `POST` | Admin | Recarga repositorios activos con el motor destino. |
| `/api/tunnel/start` | `POST` | Admin | Inicializa el túnel inverso Ngrok. |
| `/api/tunnel/stop` | `POST` | Admin | Cierra el túnel inverso. |
| `/api/audit/security` | `GET` | Admin | Consulta registros de seguridad y accesos administrativos. |
| `/api/audit/alerts` | `GET` | Admin | Consulta registros de despacho y SLA de alertas. |
