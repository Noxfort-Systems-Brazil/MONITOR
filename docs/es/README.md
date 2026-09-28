<div align="center">

<img src="../assets/monitor-logo.png" alt="Logotipo de Noxfort Monitor" width="120" />

# Noxfort Monitor™ — Suite de Documentación Técnica
### Arquitectura del Sistema, Ingestión Telemétrica y Persistencia Dual-Engine
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Activo-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Plataforma](https://img.shields.io/badge/Plataforma-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![Licencia](https://img.shields.io/badge/Licencia-AGPL_v3-blue?style=flat)](../../LICENSE)

---

🌐 **Traducciones / Idiomas:** **[🇺🇸 English](../en/README.md)** • **[🇧🇷 Português](../pt-br/README.md)** • **[🇪🇸 Español](README.md)** • **[🇫🇷 Français](../fr/README.md)** • **[🇷🇺 Русский](../ru/README.md)** • **[🇨🇳 简体中文](../zh/README.md)** • **[📚 Hub Central](../README.md)**

---

</div>

## Bienvenido a la Documentación Técnica Oficial

Este directorio contiene la suite de documentación técnica en **Español** para **Noxfort Monitor™ v2.0** — plataforma industrial de observabilidad orientada a eventos, ingestión de telemetría en tiempo real y orquestación de respuesta a incidentes desarrollada en Go para sistemas distribuidos, agentes autónomos (**Carina**, **Synapse**) y nodos IoT.

## Índice de Guías Especializadas

| Documento | Tema & Alcance | Temas Principales |
|---|---|---|
| 📖 **[Arquitectura del Sistema](architecture.md)** | Especificación de Arquitectura | Arquitectura Orientada a Eventos (EDA), concurrencia mediante Goroutines, capas SOLID e inyección de dependencias en la raíz. |
| 📡 **[API y Protocolos de Ingestión](api_reference.md)** | Protocolos de Red | Ingestión MQTT asíncrona (`:1883`), REST `POST /api/telemetry` (`:22100`), esquema JSON unificado `IncomingEvent` y rutas administrativas. |
| 🗄️ **[Base de Datos y Dual-Engine](database.md)** | Persistencia Relacional | Cambio dinámico en caliente mediante `DBManager`, PostgreSQL & SQLite puro sin CGO, migración consistente y adaptadores de dialecto. |
| 🔐 **[Seguridad, Autenticación y RBAC](security.md)** | Control de Acceso | Control de acceso basado en roles (`RoleAdmin`, `RoleOperator`), hash con sal criptográfica, cookies de sesión y arranque idempotente. |
| 🌐 **[Acceso Remoto y Túnel Ngrok](remote_access.md)** | Conectividad WAN | Túnel inverso TLS a través de cortafuegos industriales y CGNAT, dominios estáticos e ingestión telemétrica externa. |
| 🖥️ **[Aplicación Desktop y Headless](desktop_app.md)** | Interfaz y Operación | Runtime nativo en Wails v2 con WebKitGTK, bloqueo de instancia única mediante socket IPC Unix, bandeja del sistema y modo `--headless`. |
| 🔍 **[Pista de Auditoría y SLA](audit_trail.md)** | Trazabilidad y Cumplimiento | Modelo de triple auditoría: registros de seguridad (`SecurityAuditLog`), cumplimiento de SLA de alertas (`AlertDispatchLog`) e indisponibilidad. |
| 🚀 **[Guía de Despliegue en Producción](deployment.md)** | Operaciones y DevOps | Servicio systemd en modo headless, proxy inverso NGINX con terminación SSL y empaquetado Debian `.deb`. |
| 🧪 **[Pruebas y Calidad de Software](testing.md)** | Validación y Calidad | Suite de pruebas unitarias con mocks en Go, verificación manual E2E con `mosquitto_pub` / `curl` y diagnóstico de canales. |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Telemetría Industrial & Observabilidad • Noxfort Monitor™ Server v2.0</i>
</div>
