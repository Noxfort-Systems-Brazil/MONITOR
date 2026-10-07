# 👨‍💻 Guía Avanzada para Desarrolladores: Noxfort Monitor™

Este documento proporciona pautas de ingeniería de software para compilar, mantener y extender **Noxfort Monitor™ v2.0**, cubriendo configuración del entorno local, arquitectura **SOLID**, repositorios con recarga en caliente (**hot-reload**), inyección de dependencias y concurrencia con goroutines.

⬅️ [Centro de Documentación](README.md) | 🏛️ [Arquitectura](architecture.md) | 🧪 [Pruebas](testing.md) | 🗄️ [Base de Datos](database.md)

---

## 1. Entorno Local y Compilación

### Comandos Esenciales de Makefile

| Comando | Descripción |
| :--- | :--- |
| `make build` | Compila el binario de escritorio con etiquetas de producción (`bin/noxfort-monitor`). |
| `make run` | Ejecuta la aplicación completa en modo desarrollo con interfaz gráfica. |
| `make run-headless` | Ejecuta el servidor en modo Headless (sin ventana gráfica). |
| `make test` | Ejecuta la suite completa de pruebas automatizadas (Go + Vitest). |
| `make deb` | Genera el paquete de distribución Debian (`noxfort-monitor_2.0.1_amd64.deb`). |
| `make install-deb` | Instala o actualiza el paquete `.deb` generado mediante `apt`. |
| `make uninstall-deb`| Desinstala el paquete `noxfort-monitor` del sistema. |
| `make broker-start` | Inicia el broker MQTT nativo (Mosquitto). |
| `make broker-stop` | Detiene el broker MQTT. |

---

## 2. Patrones de Diseño y Arquitectura

* **Raíz de Composición**: Toda la instanciación ocurre explícitamente en [`cmd/server/main.go`](../../cmd/server/main.go). Sin variables globales ni funciones `init()`.
* **Inyección de Dependencias (DI)**: Los constructores reciben interfaces para facilitar pruebas unitarias desacopladas con mocks en memoria.
* **Patrón Repositorio con Hot-Reload**: Los repositorios implementan `ReloadableRepository` para permitir el cambio dinámico entre SQLite y PostgreSQL en caliente via [`DBManager`](../../internal/storage/db_manager.go).
