# 🏛️ Noxfort Monitor™: Arquitectura del Sistema y Concurrencia

Este documento especifica la arquitectura interna de **Noxfort Monitor™ v2.0**, detallando la Arquitectura Orientada a Eventos (EDA), el modelo de concurrencia con Goroutines, la delimitación de capas SOLID y el ensamblaje en la raíz de composición.

⬅️ [Hub Central](../README.md) | 📡 [API y Protocolos](api_reference.md) | 🗄️ [Base de Datos](database.md) | 🔐 [Seguridad](security.md) | 🧪 [Pruebas](testing.md)

---

## 1. Filosofía Arquitectónica

Noxfort Monitor se rige por una estricta **Arquitectura Orientada a Eventos (EDA)** acoplada a principios **SOLID** y a la **Inyección de Dependencias (DI)** articulada en `cmd/server/main.go`. El sistema prescinde de estados globales mutables y paquetes con rutinas de inicialización opacas.

```mermaid
graph TD
    subgraph "Mundo Externo & Nodos de Borde"
        LocalDevice[Dispositivo Local / LAN]
        RemoteDevice[Agente Remoto / WAN (Carina, Synapse)]
        Operator[Navegador / Operador Humano]
    end

    subgraph "Capa de Transporte & Red"
        MQTT[Broker MQTT :1883]
        Ngrok[Túnel Ngrok / WAN HTTPS]
        HTTP[Servidor HTTP :22100]
        AuthMW[AuthMiddleware - RBAC]
    end

    subgraph "Capa de Lógica & Seguridad"
        StateManager[Gestor de Estado]
        Watchdog[Motor Watchdog]
        Alerts[Servicio de Enrutamiento de Alertas]
        SecManager[Gestor de Seguridad]
        TunnelMgr[Gestor de Túnel]
    end

    subgraph "Persistencia Dual-Engine (internal/storage)"
        DBMgr[DBManager Central]
        AuditRepo[AuditRepository]
        PG[(PostgreSQL Industrial)]
        SQLite[(SQLite Puro sin CGO)]
    end

    subgraph "Canales Externos de Notificación"
        Telegram[Telegram Bot API (MarkdownV2)]
        Email[Servidor SMTP (E-mail)]
    end

    LocalDevice -- "Publicación MQTT" --> MQTT
    RemoteDevice -- "HTTPS POST /api/telemetry" --> Ngrok
    Ngrok --> HTTP
    Operator -- "HTTP GET / POST" --> HTTP

    MQTT -- "Decodifica Payload" --> StateManager
    HTTP --> AuthMW
    AuthMW --> StateManager
    AuthMW --> SecManager

    StateManager -- "1. Persiste Incidente" --> DBMgr
    StateManager -- "2. Despacha Alerta" --> Alerts
    Watchdog -- "Verifica Latidos" --> DBMgr
    Watchdog -- "Sintetiza Caída/Recuperación" --> Alerts
    Watchdog -- "Registra Transición" --> AuditRepo

    Alerts -- "Goroutines Concurrentes" --> Telegram
    Alerts -- "Goroutines Concurrentes" --> Email
    Alerts -- "Registra SLA de Despacho" --> AuditRepo

    SecManager -- "Audita Inicios de Sesión" --> AuditRepo
    DBMgr --> PG
    DBMgr --> SQLite
```

---

## 2. Capas de Subsistemas Desacopladas

1. **Capa de Transporte (`internal/transport`)**: Terminación de protocolos de red (broker MQTT mediante Paho y servidor HTTP REST con `AuthMiddleware`).
2. **Capa de Lógica de Monitorización (`internal/monitor`)**: Máquina de estados reactiva (`StateManager`), detección de fallos silenciosos (`Engine`/Watchdog) y despacho de alertas (`AlertService`).
3. **Capa de Seguridad (`internal/security`)**: Gestión de sesiones, hash de contraseñas con sal criptográfica, validación de tokens y control de acceso basado en roles (RBAC).
4. **Capa de Acceso Remoto (`internal/tunnel`)**: Túnel inverso seguro mediante Ngrok para ingestión de nodos de borde en redes WAN.
5. **Capa de Dominio (`internal/domain`)**: Modelos de datos puros y contratos de interfaz sin dependencias externas.
6. **Capa de Persistencia (`internal/storage`)**: Administrador dual-engine dinámico (`DBManager`), controladores PostgreSQL y SQLite, adaptador de dialectos y migrador de datos.
7. **Capa de Interfaz Desktop (`internal/desktop`, `internal/tray`)**: Runtime nativo en Wails v2 con WebKitGTK, bloqueo de instancia única mediante socket Unix e integración de bandeja del sistema.

---

## 3. Concurrencia y Modelo de Goroutines

```mermaid
graph TD
    Main[Hilo Principal del SO / Goroutine Primaria]
    
    Main -->|Modo Desktop Estándar| WailsEventLoop[Bucle de Eventos Wails v2]
    WailsEventLoop --> Systray[Callbacks GTK de Bandeja]
    WailsEventLoop --> WebKit[Ventana WebKitGTK]
    
    Main -->|Modo --headless| SigChan[Bucle de Señales del SO (SIGINT/SIGTERM)]

    Main -.->|go func| HTTPServer[Servidor HTTP ListenAndServe :22100]
    Main -.->|go func| MQTTListener[Bucle de Paquetes MQTT Paho]
    Main -.->|go func| WatchdogEngine[Bucle del Ticker - Intervalo 30s]
    Main -.->|go func| AlertWorkers[Workers Concurrentes Email/Telegram]
```

* **Hilo Principal**: En modo desktop ejecuta `desktopApp.Run()` (Wails v2). En modo `--headless` bloquea esperando señales del sistema operativo (`syscall.SIGTERM`, `os.Interrupt`).
* **Servidor HTTP**: Corre en goroutine independiente con límites de tiempo de 15 segundos en el puerto `22100`.
* **Listener MQTT**: Paho gestiona sockets TCP en goroutines dedicadas de lectura/escritura en el puerto `1883`.
* **Bucle Watchdog**: Canal desacoplado con `time.Ticker` que evalúa ausencia de transmisiones (> 5 minutos).
* **Cierre Limpio**: Coordinado por una rutina `sync.Once` thread-safe que libera sockets de red y conexiones a bases de datos de forma atómica.
