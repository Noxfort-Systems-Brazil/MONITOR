# 🔬 Notes de Recherche & Décisions Techniques : Noxfort Monitor™

Ce document consigne les décisions d'architecture (ADR), benchmarks et axes de recherche de **Noxfort Monitor™**.

⬅️ [Centre de Documentation](README.md) | 🏛️ [Architecture](architecture.md) | 🗄️ [Base de Données](database.md) | 🌐 [Accès Distant](remote_access.md)

---

## 1. Persistance : De CGO à l'Architecture Dual-Engine
* **Migration vers Pure-Go SQLite** : Adoption de `modernc.org/sqlite` sans dépendance CGO.
* **Dual-Engine (SQLite + PostgreSQL)** : Basculement dynamique à chaud via [`DBManager`](database.md).

## 2. Interface Bureau : De Systray à Wails v2
* **WebKitGTK Natif** : Rendu rapide et léger sans la surcharge mémoire de Chromium/Electron.
* **Mode Headless** : Option `--headless` pour serveurs Linux sans serveur d'affichage X11/Wayland.

## 3. Connectivité WAN : DuckDNS + Caddy vs Tunnel Ngrok
* **DuckDNS + Caddy** : Solution primaire recommandée avec certificats ACME DNS-01 automatiques sur port 443.
* **Tunnel Ngrok** : Solution alternative pour environnements avec CGNAT strict.

## 4. Feuille de Route
* **Fédération gRPC** : Réplication typée haute performance pour clusters distribués.
* **Détection Prédictive des Anomalies** : Analyse temporelle pour anticiper les pannes matérielles.
