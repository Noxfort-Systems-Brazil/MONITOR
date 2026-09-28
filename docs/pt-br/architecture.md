# 🏛️ Noxfort Monitor™: Blueprint e Arquitetura do Sistema

Este documento especifica a arquitetura interna do **Noxfort Monitor™ v2.0**, detalhando a Arquitetura Orientada a Eventos (EDA), o modelo de concorrência com Goroutines, os limites de camadas SOLID e a montagem na raiz de composição.

⬅️ [Central de Documentação](../README.md) | 📡 [Referência de API](api_reference.md) | 🗄️ [Banco de Dados](database.md) | 🔐 [Segurança](security.md) | 🧪 [Testes](testing.md)

---

## 1. Filosofia Arquitetural

O Noxfort Monitor opera sob uma estrita **Arquitetura Orientada a Eventos (EDA)** acoplada a princípios **SOLID** e **Injeção de Dependências (DI)** montada no ponto de composição em `cmd/server/main.go`. O sistema elimina estados globais mutáveis e pacotes com rotinas de inicialização ocultas.

```mermaid
graph TD
    subgraph "Mundo Externo & Nós de Borda"
        LocalDevice[Dispositivo Local / LAN]
        RemoteDevice[Agente Remoto / WAN (Carina, Synapse)]
        Operator[Navegador / Operador Humano]
    end

    subgraph "Camada de Transporte & Rede"
        MQTT[Broker MQTT :1883]
        Ngrok[Túnel Ngrok / WAN HTTPS]
        HTTP[Servidor HTTP :22100]
        AuthMW[AuthMiddleware - RBAC]
    end

    subgraph "Camada de Lógica & Segurança"
        StateManager[Gerenciador de Estado]
        Watchdog[Motor Watchdog]
        Alerts[Serviço de Roteamento de Alertas]
        SecManager[Gerenciador de Segurança]
        TunnelMgr[Gerenciador de Túnel]
    end

    subgraph "Persistência Dual-Engine (internal/storage)"
        DBMgr[DBManager Central]
        AuditRepo[AuditRepository]
        PG[(PostgreSQL Industrial)]
        SQLite[(SQLite Puro sem CGO)]
    end

    subgraph "Canais Externos de Notificação"
        Telegram[Telegram Bot API (MarkdownV2)]
        Email[Servidor SMTP (E-mail)]
    end

    LocalDevice -- "Publicação MQTT" --> MQTT
    RemoteDevice -- "HTTPS POST /api/telemetry" --> Ngrok
    Ngrok --> HTTP
    Operator -- "HTTP GET / POST" --> HTTP

    MQTT -- "Decodifica Payload" --> StateManager
    HTTP --> AuthMW
    AuthMW --> StateManager
    AuthMW --> SecManager

    StateManager -- "1. Persiste Incidente" --> DBMgr
    StateManager -- "2. Despacha Alerta" --> Alerts
    Watchdog -- "Verifica Batimentos" --> DBMgr
    Watchdog -- "Sintetiza Queda/Recuperação" --> Alerts
    Watchdog -- "Registra Transição" --> AuditRepo

    Alerts -- "Goroutines Concorrentes" --> Telegram
    Alerts -- "Goroutines Concorrentes" --> Email
    Alerts -- "Registra SLA de Despacho" --> AuditRepo

    SecManager -- "Audita Logins" --> AuditRepo
    DBMgr --> PG
    DBMgr --> SQLite
```

---

## 2. Camadas de Subsistemas Desacopladas

1. **Camada de Transporte (`internal/transport`)**: Terminação de protocolos de rede (broker MQTT via Paho e servidor HTTP REST com `AuthMiddleware`).
2. **Camada de Lógica de Monitoramento (`internal/monitor`)**: Máquina de estados reativa (`StateManager`), detector de falhas silenciosas (`Engine`/Watchdog) e roteador de alertas (`AlertService`).
3. **Camada de Segurança (`internal/security`)**: Gerenciamento de sessões, hash de senhas com sal, validação de tokens e controle de acesso baseado em papéis (RBAC).
4. **Camada de Acesso Remoto (`internal/tunnel`)**: Túnel reverso seguro via Ngrok para ingestão de nós de borda em redes WAN.
5. **Camada de Domínio (`internal/domain`)**: Entidades e contratos de interface puros sem dependências externas.
6. **Camada de Persistência (`internal/storage`)**: Gerenciador dinâmico dual-engine (`DBManager`), implementações PostgreSQL e SQLite, adaptador de dialeto e migrador de dados.
7. **Camada de Interface Desktop (`internal/desktop`, `internal/tray`)**: Runtime nativo em Wails v2 com WebKitGTK, trava de instância única via socket Unix e ciclo de vida da bandeja do sistema.

---

## 3. Modelo de Concorrência e Goroutines

```mermaid
graph TD
    Main[Thread Principal do SO / Goroutine Primária]
    
    Main -->|Modo Desktop Padrão| WailsEventLoop[Loop de Eventos Desktop Wails v2]
    WailsEventLoop --> Systray[Callbacks GTK da Bandeja]
    WailsEventLoop --> WebKit[Janela WebKitGTK]
    
    Main -->|Modo --headless| SigChan[Loop de Sinais do SO (SIGINT/SIGTERM)]

    Main -.->|go func| HTTPServer[Servidor HTTP ListenAndServe :22100]
    Main -.->|go func| MQTTListener[Loop de Pacotes MQTT Paho]
    Main -.->|go func| WatchdogEngine[Loop do Ticker - Intervalo 30s]
    Main -.->|go func| AlertWorkers[Workers Concorrentes de Email/Telegram]
```

* **Thread Principal**: No modo desktop, executa `desktopApp.Run()` (Wails v2), exigido porque o toolkit GTK/WebKit do Linux deve deter a thread primária do SO. No modo `--headless`, bloqueia aguardando sinais de finalização do SO (`syscall.SIGTERM`, `os.Interrupt`).
* **Servidor HTTP**: Executado em goroutine independente com timeouts de 15 segundos na porta `22100`.
* **Listener MQTT**: O cliente Paho gerencia sockets TCP em goroutines dedicadas de leitura/escrita na porta `1883`.
* **Loop do Watchdog**: Opera em canal `time.Ticker` desacoplado avaliando ausência de batimento (> 5 minutos).
* **Encerramento Gracioso**: Coordenado por rotina `sync.Once` thread-safe que encerra sockets de rede, fecha túneis e libera pools de banco de forma atômica.
