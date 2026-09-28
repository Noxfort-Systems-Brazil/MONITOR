# 🏛️ Noxfort Monitor™: Архитектура системы и параллелизм

Данный документ описывает внутреннюю архитектуру **Noxfort Monitor™ v2.0**, включая событийно-ориентированную модель (EDA), модель параллелизма на горутинах, разделение на слои SOLID и сборку в корне композиции.

⬅️ [Главный Хаб](../README.md) | 📡 [Справочник API](api_reference.md) | 🗄️ [База данных](database.md) | 🔐 [Безопасность](security.md) | 🧪 [Тестирование](testing.md)

---

## 1. Философия архитектуры

Noxfort Monitor построен на базе строгой **Событийно-Ориентированной Архитектуры (EDA)** в сочетании с принципами **SOLID** и **Внедрением Зависимостей (DI)** в точке входа `cmd/server/main.go`. В системе полностью исключено разделяемое изменяемое глобальное состояние.

```mermaid
graph TD
    subgraph "Внешний мир & Полевые узлы"
        LocalDevice[Локальное устройство / LAN]
        RemoteDevice[Удаленный агент / WAN (Carina, Synapse)]
        Operator[Браузер / Оператор]
    end

    subgraph "Транспортный и сетевой уровень"
        MQTT[Брокер MQTT :1883]
        Ngrok[Туннель Ngrok / WAN HTTPS]
        HTTP[HTTP-сервер :22100]
        AuthMW[AuthMiddleware - RBAC]
    end

    subgraph "Уровень логики и безопасности"
        StateManager[Менеджер состояния]
        Watchdog[Движок Watchdog]
        Alerts[Маршрутизатор оповещений]
        SecManager[Менеджер безопасности]
        TunnelMgr[Менеджер туннеля]
    end

    subgraph "Двухдвижковая персистентность (internal/storage)"
        DBMgr[Центральный DBManager]
        AuditRepo[AuditRepository]
        PG[(Промышленный PostgreSQL)]
        SQLite[(Чистый SQLite без CGO)]
    end

    subgraph "Внешние каналы уведомлений"
        Telegram[Telegram Bot API (MarkdownV2)]
        Email[SMTP-сервер (E-mail)]
    end

    LocalDevice -- "MQTT Publish" --> MQTT
    RemoteDevice -- "HTTPS POST /api/telemetry" --> Ngrok
    Ngrok --> HTTP
    Operator -- "HTTP GET / POST" --> HTTP

    MQTT -- "Декодирует Payload" --> StateManager
    HTTP --> AuthMW
    AuthMW --> StateManager
    AuthMW --> SecManager

    StateManager -- "1. Сохраняет инцидент" --> DBMgr
    StateManager -- "2. Отправляет оповещение" --> Alerts
    Watchdog -- "Проверяет Heartbeats" --> DBMgr
    Watchdog -- "Синтезирует отказ/восстановление" --> Alerts
    Watchdog -- "Записывает переход" --> AuditRepo

    Alerts -- "Параллельные горутины" --> Telegram
    Alerts -- "Параллельные горутины" --> Email
    Alerts -- "Фиксирует SLA отправки" --> AuditRepo

    SecManager -- "Аудит авторизации" --> AuditRepo
    DBMgr --> PG
    DBMgr --> SQLite
```

---

## 2. Изоляция подсистем по слоям

1. **Транспортный уровень (`internal/transport`)**: Завершение протоколов (клиент MQTT Paho и HTTP REST с `AuthMiddleware`).
2. **Уровень логики мониторинга (`internal/monitor`)**: Реактивный конечный автомат (`StateManager`), детектор скрытых отказов (`Engine`/Watchdog) и диспетчер тревог (`AlertService`).
3. **Уровень безопасности (`internal/security`)**: Управление сессиями, криптографическое хеширование с солью и ролевой доступ (RBAC).
4. **Уровень удаленного доступа (`internal/tunnel`)**: Защищенный обратный туннель Ngrok для приема телеметрии из сетей WAN.
5. **Уровень домена (`internal/domain`)**: Чистые структуры данных и интерфейсные контракты без внешних зависимостей.
6. **Уровень хранения (`internal/storage`)**: Динамический менеджер `DBManager`, драйверы PostgreSQL и SQLite, адаптер диалектов и мигратор.
7. **Уровень десктопа (`internal/desktop`, `internal/tray`)**: Wails v2 + WebKitGTK, блокировка повторного запуска через Unix-сокет и трей.

---

## 3. Модель параллелизма на горутинах

```mermaid
graph TD
    Main[Главный поток ОС / Основная горутина]
    
    Main -->|Десктопный режим| WailsEventLoop[Цикл событий Wails v2]
    WailsEventLoop --> Systray[Коллбэки трея GTK]
    WailsEventLoop --> WebKit[Окно WebKitGTK]
    
    Main -->|Режим --headless| SigChan[Цикл обработки сигналов (SIGINT/SIGTERM)]

    Main -.->|go func| HTTPServer[HTTP-сервер ListenAndServe :22100]
    Main -.->|go func| MQTTListener[Цикл пакетов MQTT Paho]
    Main -.->|go func| WatchdogEngine[Цикл тикера - 30с]
    Main -.->|go func| AlertWorkers[Воркеры рассылки Email/Telegram]
```

* **Главный поток**: В десктопном режиме выполняет `desktopApp.Run()`. В режиме `--headless` блокируется в ожидании сигналов ОС (`syscall.SIGTERM`, `os.Interrupt`).
* **HTTP-сервер**: Работает в независимой горутине на порту `22100` с таймаутами 15 секунд.
* **MQTT-слушатель**: Клиент Paho управляет TCP-сокетами в отдельных горутинах ввода-вывода на порту `1883`.
* **Цикл Watchdog**: Работает по каналу `time.Ticker`, фиксируя молчание оборудования (> 5 минут).
* **Корректное завершение**: Потокобезопасная процедура `sync.Once` закрывает сокеты и освобождает пулы подключений.
