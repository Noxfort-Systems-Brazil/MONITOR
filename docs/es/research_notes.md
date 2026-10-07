# 🔬 Notas de Investigación y Decisiones Técnicas: Noxfort Monitor™

Este documento consolida el registro de decisiones arquitectónicas (ADRs), puntos de referencia y la hoja de ruta de **Noxfort Monitor™**.

⬅️ [Centro de Documentación](README.md) | 🏛️ [Arquitectura](architecture.md) | 🗄️ [Base de Datos](database.md) | 🌐 [Acceso Remoto](remote_access.md)

---

## 1. Persistencia: De CGO a la Arquitectura Dual-Engine
* **Migración a Pure-Go SQLite**: Adopción de `modernc.org/sqlite` para compilar sin dependencias de CGO.
* **Dual-Engine (SQLite + PostgreSQL)**: Soporte dinámico para despliegues ligeros y clusters empresariales de alta concurrencia con hot-reload en caliente via `DBManager`.

## 2. Interfaz de Escritorio: De Systray a Wails v2
* **WebKitGTK Nativo**: Renderizado rápido y consistente sin la sobrecarga de memoria de Chromium/Electron.
* **Compatibilidad Headless**: Modo `--headless` para servidores Linux sin entorno gráfico X11/Wayland.

## 3. Conectividad WAN: DuckDNS + Caddy vs Túnel Ngrok
* **DuckDNS + Caddy**: Estrategia primaria con certificados SSL/TLS automáticos via ACME DNS-01 y puerto 443.
* **Túnel Ngrok**: Alternativa de contingencia para redes con CGNAT estricto.

## 4. Hoja de Ruta Futura
* **Federación con gRPC**: Replicación eficiente y tipada entre servidores distribuidos.
* **Detección Predictiva de Anomalías**: Modelos de series temporales para predecir fallas antes de alertas críticas.
