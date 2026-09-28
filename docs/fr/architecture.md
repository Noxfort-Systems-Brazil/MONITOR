# 🏛️ Noxfort Monitor™ : Architecture Système et Concurrence

Ce document spécifie l'architecture interne de **Noxfort Monitor™ v2.0**, détaillant l'architecture orientée événements (EDA), la concurrence multithread via Goroutines, le découpage en couches SOLID et l'assemblage racine.

⬅️ [Hub Central](../README.md) | 📡 [API Réseau](api_reference.md) | 🗄️ [Base de Données](database.md) | 🔐 [Sécurité](security.md) | 🧪 [Tests](testing.md)

---

## 1. Philosophie d'Architecture

Noxfort Monitor repose sur une **Architecture Orientée Événements (EDA)** rigoureuse couplée aux principes **SOLID** et à l'**Injection de Dépendances (DI)** assemblée dans `cmd/server/main.go`. Le système élimine tout état global mutable partagé et tout paquet à initialisation opaque.

```mermaid
graph TD
    subgraph "Monde Extérieur & Nœuds Distants"
        LocalDevice[Périphérique Local / LAN]
        RemoteDevice[Agent Distant / WAN (Carina, Synapse)]
        Operator[Navigateur / Opérateur Humain]
    end

    subgraph "Couche Réseau & Transport"
        MQTT[Broker MQTT :1883]
        Ngrok[Tunnel Ngrok / WAN HTTPS]
        HTTP[Serveur HTTP :22100]
        AuthMW[AuthMiddleware - RBAC]
    end

    subgraph "Couche Logique & Sécurité"
        StateManager[Gestionnaire d'État]
        Watchdog[Moteur Watchdog]
        Alerts[Service de Routage d'Alertes]
        SecManager[Gestionnaire de Sécurité]
        TunnelMgr[Gestionnaire de Tunnel]
    end

    subgraph "Persistance Double-Moteur (internal/storage)"
        DBMgr[DBManager Central]
        AuditRepo[AuditRepository]
        PG[(PostgreSQL Industriel)]
        SQLite[(SQLite Pur sans CGO)]
    end

    subgraph "Canaux de Notification Externes"
        Telegram[Telegram Bot API (MarkdownV2)]
        Email[Serveur SMTP (E-mail)]
    end

    LocalDevice -- "Publication MQTT" --> MQTT
    RemoteDevice -- "HTTPS POST /api/telemetry" --> Ngrok
    Ngrok --> HTTP
    Operator -- "HTTP GET / POST" --> HTTP

    MQTT -- "Décode Payload" --> StateManager
    HTTP --> AuthMW
    AuthMW --> StateManager
    AuthMW --> SecManager

    StateManager -- "1. Persiste Incident" --> DBMgr
    StateManager -- "2. Déclenche Alerte" --> Alerts
    Watchdog -- "Vérifie Battements" --> DBMgr
    Watchdog -- "Synthétise Panne/Rétablissement" --> Alerts
    Watchdog -- "Enregistre Transition" --> AuditRepo

    Alerts -- "Goroutines Concurrentes" --> Telegram
    Alerts -- "Goroutines Concurrentes" --> Email
    Alerts -- "Enregistre SLA d'Envoi" --> AuditRepo

    SecManager -- "Audite Connexions" --> AuditRepo
    DBMgr --> PG
    DBMgr --> SQLite
```

---

## 2. Découpage Modulaire des Couches

1. **Couche Transport (`internal/transport`)** : Terminaison des protocoles réseau (broker MQTT Paho et serveur HTTP REST avec `AuthMiddleware`).
2. **Couche Logique de Surveillance (`internal/monitor`)** : Machine à états réactive (`StateManager`), détection de pannes silencieuses (`Engine`/Watchdog) et routage d'alertes (`AlertService`).
3. **Couche Sécurité (`internal/security`)** : Gestion des sessions, hachage de mots de passe avec sel, jetons d'accès et contrôle d'accès basé sur les rôles (RBAC).
4. **Couche Accès Distant (`internal/tunnel`)** : Tunnel inverse sécurisé via Ngrok pour l'ingestion de nœuds de terrain sur les réseaux WAN.
5. **Couche Domaine (`internal/domain`)** : Modèles de données purs et contrats d'interfaces sans dépendance externe.
6. **Couche Persistance (`internal/storage`)** : Gestionnaire double-moteur dynamique (`DBManager`), pilotes PostgreSQL et SQLite, adaptateur de dialectes et migration de données.
7. **Couche Desktop (`internal/desktop`, `internal/tray`)** : Runtime natif en Wails v2 avec WebKitGTK, exclusion mutuelle via socket Unix et intégration de la barre des tâches.

---

## 3. Concurrence et Modèle de Goroutines

```mermaid
graph TD
    Main[Thread Principal de l'OS / Goroutine Racine]
    
    Main -->|Mode Desktop Standard| WailsEventLoop[Boucle d'Événements Wails v2]
    WailsEventLoop --> Systray[Callbacks GTK Systray]
    WailsEventLoop --> WebKit[Fenêtre WebKitGTK]
    
    Main -->|Mode --headless| SigChan[Boucle de Signaux OS (SIGINT/SIGTERM)]

    Main -.->|go func| HTTPServer[Serveur HTTP ListenAndServe :22100]
    Main -.->|go func| MQTTListener[Boucle de Paquets MQTT Paho]
    Main -.->|go func| WatchdogEngine[Boucle du Ticker - Intervalle 30s]
    Main -.->|go func| AlertWorkers[Workers Concurrents Email/Telegram]
```

* **Thread Principal** : En mode desktop, exécute `desktopApp.Run()` (Wails v2). En mode `--headless`, bloque sur un canal de signaux de terminaison (`syscall.SIGTERM`, `os.Interrupt`).
* **Serveur HTTP** : Exécuté dans une goroutine indépendante sur le port `22100` avec des délais de 15 secondes.
* **Écouteur MQTT** : Gère les sockets TCP sur des goroutines d'E/S dédiées sur le port `1883`.
* **Surveillance Watchdog** : Évalue les battements de cœur sur un canal cadencé par `time.Ticker` (> 5 minutes de silence).
* **Arrêt Contrôlé** : Coordonné par une routine thread-safe `sync.Once` libérant tous les sockets et pools de connexions proprement.
