<div align="center">

<img src="../assets/monitor-logo.png" alt="Logo de Noxfort Monitor" width="120" />

# Noxfort Monitor™ — Suite de Documentation Technique
### Architecture Système, Ingestion Télémétrique Zero-Trust et Persistance Double-Moteur
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Actif-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Plateforme](https://img.shields.io/badge/Plateforme-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![Licence: AGPL-3.0-or-later](https://img.shields.io/badge/Licence-AGPL--3.0--or--later-blue.svg)](../../LICENSE)

---

🌐 **Traductions / Langues :** **[🇺🇸 English](../en/README.md)** • **[🇧🇷 Português](../pt-br/README.md)** • **[🇪🇸 Español](../es/README.md)** • **[🇫🇷 Français](README.md)** • **[🇷🇺 Русский](../ru/README.md)** • **[🇨🇳 简体中文](../zh/README.md)** • **[📚 Hub Central](../README.md)**

---

</div>

## Bienvenue dans la Documentation Technique Officielle

Ce répertoire rassemble la suite documentaire technique en **Français** pour **Noxfort Monitor™ v2.0** — plateforme industrielle d'observabilité événementielle, d'ingestion télémétrique Zero-Trust en temps réel et d'orchestration de réponse aux incidents conçue en Go pour les systèmes distribués, les agents autonomes (**Carina**, **Synapse**) et les nœuds IoT.

## Répertoire des Guides Techniques

| Document | Sujet & Périmètre | Contenu Principal |
|---|---|---|
| 📖 **[Architecture Système](architecture.md)** | Spécification d'Architecture | Architecture orientée événements (EDA), Goroutines, `BufferedTelemetryWriter`, modes Desktop / `--headless` / `--demo` et arrêt propre (`AlertService.Shutdown`). |
| 📡 **[API et Protocoles d'Ingestion](api_reference.md)** | Protocoles Réseau | MQTT asynchrone `noxfort/telemetry/#` (`MONITOR_MQTT_STRICT`), appairage Zero-Trust `POST /api/pair` et `POST /api/telemetry` (`:22100`), schéma `IncomingEvent` et routes RBAC. |
| 🗄️ **[Base de Données et Persistance](database.md)** | Persistance Double-Moteur | SQLite WAL + PostgreSQL avec basculement à chaud via `DBManager`, `7` `ReloadableRepository`, `MigrateData` transactionnel (9 tables), sauvegardes/restauration et `RetentionManager`. |
| 🔐 **[Sécurité, Authentification et RBAC](security.md)** | Sécurité et Contrôle d'Accès | `OPERATOR` strictement en lecture seule (`403` / `forbidden.html`), `PBKDF2Hasher` (100k itér.), `LoginAttemptTracker` (`429`), `DBSessionStore` (`SHA-256`), `TRUSTED_PROXIES` et masquage HTML. |
| 🌐 **[Accès Distant et WAN (DuckDNS & Ngrok)](remote_access.md)** | Connectivité WAN | Synchronisation dynamique DuckDNS, tunnel inverse Ngrok, rédaction des jetons dans `/remote` et ingestion Zero-Trust distante. |
| 🛡️ **[Proxy Inverse Caddy & DuckDNS](caddy_integration.md)** | Passerelle HTTPS | Terminaison TLS/SSL ACME DNS-01, en-têtes `X-Forwarded-*` avec `TRUSTED_PROXIES`, port 443 et WebSockets chiffrés (`/mqtt`). |
| 🖥️ **[Application Desktop, Headless et Demo](desktop_app.md)** | Interface et Exploitation | Runtime Wails v2 + WebKitGTK, verrou d'instance unique IPC Unix, modes `--headless` et `--demo`, polling `/api/dashboard/summary` et page `forbidden.html`. |
| 🔍 **[Piste d'Audit et SLA](audit_trail.md)** | Traçabilité et Conformité | Triple audit (`SecurityAuditLog`, `AlertDispatchLog` lié à `telemetry_id`, `DeviceStateTransition`), lecture `Operator+` et purge automatique après 90 jours. |
| 🚀 **[Guide de Déploiement en Production](deployment.md)** | Exploitation et DevOps | Variables `.env` (`configs/config.yaml` supprimé), systemd headless, Docker Compose, Caddy/NGINX, paquet `.deb`, sauvegardes et `/metrics`. |
| 🧪 **[Tests et Assurance Qualité](testing.md)** | Validation et Qualité | Suite Go + Vitest (`make test`), mocks en mémoire, tests E2E `mosquitto_pub` (`noxfort/telemetry/#`) / `curl` (`/api/pair` + `X-Device-Token`) et `make record-demo`. |
| 👨‍💻 **[Guide Développeur Avancé](developer_guides.md)** | Ingénierie et Extensibilité | Environnement local, contrat `ReloadableRepository` (`SetDB(db, driver)`), `internal/metrics`, utilitaires HTTP de sécurité et cycle `bgCtx`. |
| 🔬 **[Notes de Recherche & Décisions Techniques](research_notes.md)** | Registre d'Architecture (ADR) | Décisions techniques : suppression de CGO et `config.yaml`, appairage Zero-Trust, sessions hachées en BD, WAL/rétention et métriques Prometheus. |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Télémétrie Industrielle & Observabilité • Noxfort Monitor™ Server v2.0</i><br/>
  <small>Sous licence <a href="../../LICENSE">GNU Affero General Public License v3.0 ou ultérieure</a> (SPDX: <code>AGPL-3.0-or-later</code>). © 2026 Noxfort Systems.</small>
</div>
