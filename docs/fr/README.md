<div align="center">

<img src="../assets/monitor-logo.png" alt="Logo de Noxfort Monitor" width="120" />

# Noxfort Monitor™ — Suite de Documentation Technique
### Architecture Système, Ingestion Télémétrique et Persistance Double-Moteur
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Actif-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Plateforme](https://img.shields.io/badge/Plateforme-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![Licence](https://img.shields.io/badge/Licence-AGPL_v3-blue?style=flat)](../../LICENSE)

---

🌐 **Traductions / Langues :** **[🇺🇸 English](../en/README.md)** • **[🇧🇷 Português](../pt-br/README.md)** • **[🇪🇸 Español](../es/README.md)** • **[🇫🇷 Français](README.md)** • **[🇷🇺 Русский](../ru/README.md)** • **[🇨🇳 简体中文](../zh/README.md)** • **[📚 Hub Central](../README.md)**

---

</div>

## Bienvenue dans la Documentation Technique Officielle

Ce répertoire rassemble la suite documentaire technique en **Français** pour **Noxfort Monitor™ v2.0** — plateforme industrielle d'observabilité événementielle, d'ingestion télémétrique en temps réel et d'orchestration de réponse aux incidents conçue en Go pour les systèmes distribués, les agents autonomes (**Carina**, **Synapse**) et les nœuds IoT.

## Répertoire des Guides Techniques

| Document | Sujet & Périmètre | Contenu Principal |
|---|---|---|
| 📖 **[Architecture Système](architecture.md)** | Spécification d'Architecture | Architecture orientée événements (EDA), concurrence via Goroutines, couches SOLID et injection de dépendances à la racine. |
| 📡 **[API et Protocoles d'Ingestion](api_reference.md)** | Protocoles Réseau | Ingestion MQTT asynchrone (`:1883`), REST `POST /api/telemetry` (`:22100`), schéma JSON unifié `IncomingEvent` et routes d'administration. |
| 🗄️ **[Base de Données et Persistance](database.md)** | Persistance Double-Moteur | Basculement à chaud via `DBManager`, PostgreSQL & SQLite pur sans CGO, migration cohérente et adaptation des dialectes SQL. |
| 🔐 **[Sécurité, Authentification et RBAC](security.md)** | Sécurité et Contrôle d'Accès | Contrôle d'accès basé sur les rôles (`RoleAdmin`, `RoleOperator`), hachage avec sel, cookies de session et amorçage idempotent. |
| 🌐 **[Accès Distant et Tunnel Ngrok](remote_access.md)** | Connectivité WAN | Tunnel inversé TLS franchissant pare-feu industriels et CGNAT, domaines statiques et ingestion télémétrique externe. |
| 🖥️ **[Application Desktop et Headless](desktop_app.md)** | Interface et Exploitation | Runtime natif Wails v2 + WebKitGTK, verrou d'instance unique via socket IPC Unix, barre d'état (Systray) et mode daemon `--headless`. |
| 🔍 **[Piste d'Audit et SLA](audit_trail.md)** | Traçabilité et Conformité | Modèle d'audit triple : logs d'accès de sécurité (`SecurityAuditLog`), conformité SLA des alertes (`AlertDispatchLog`) et temps d'arrêt. |
| 🚀 **[Guide de Déploiement en Production](deployment.md)** | Exploitation et DevOps | Service systemd en mode headless, proxy inverse NGINX avec terminaison SSL et paquet Debian `.deb`. |
| 🧪 **[Tests et Assurance Qualité](testing.md)** | Validation et Qualité | Suite de tests unitaires avec mocks en Go, validation E2E manuelle avec `mosquitto_pub` / `curl` et diagnostics de canaux. |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Télémétrie Industrielle & Observabilité • Noxfort Monitor™ Server v2.0</i>
</div>
