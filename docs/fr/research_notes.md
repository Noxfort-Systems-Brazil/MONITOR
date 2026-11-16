# 🔬 Notes de Recherche & Décisions Techniques : Noxfort Monitor™

Ce document consigne les décisions d'architecture (ADR) et l'évolution technique de **Noxfort Monitor™ v2.0**.

⬅️ [Centre de Documentation](README.md) | 🏛️ [Architecture](architecture.md) | 🗄️ [Base de Données](database.md) | 🌐 [Accès Distant](remote_access.md)

---

## 1. Suppression de `configs/config.yaml` au Profit de `.env` + SQL `settings`
* **Décision** : Retrait de `configs/config.yaml`. Les variables de démarrage proviennent de `.env` (`LoadDotEnv()`), et la configuration opérationnelle (URL MQTT, alertes, tunnels) vit dans `settings` avec rechargement dynamique (`SetOnMQTTConfigChanged` $\rightarrow$ `Reconnect`).

## 2. Appairage Zero-Trust (`POST /api/pair`) & `MONITOR_MQTT_STRICT`
* **Décision** : `POST /api/pair` place en file `PENDING` (plafond 50, `409 Conflict` si déjà `ACTIVE`) et retourne `202` **sans clé** ; le jeton `nox_live_` (TTL 5m) est copié hors bande depuis `/devices` puis vérifié via `X-Device-Token`/`Bearer`, `X-Device-Origin` et fenêtre `±300s`. En MQTT, `MONITOR_MQTT_STRICT=true` rejette les nœuds inconnus ou `PAUSED` sur `noxfort/telemetry/#`.

## 3. Sessions Hachées Persistantes, Verrouillage PBKDF2 & RBAC Strict
* **Décision** : Remplacement des sessions en mémoire par `DBSessionStore` (`SHA-256` dans `sessions`, TTL 24h, `ReloadableRepository`), rejet de `?token=`, `PBKDF2Hasher` (`100 000` itérations) + `LoginAttemptTracker` (5/15m/5m $\rightarrow$ `429`), et profil `OPERATOR` strictement en lecture seule (+ mot de passe personnel).

## 4. Stockage Double-Moteur : SQLite WAL, Migration Atomique, Backups & Rétention
* **Décision** : SQLite Pure-Go (`modernc.org/sqlite`) en mode `WAL`, `MigrateData` transactionnel unique sur 9 tables, `BackupManager` (`VACUUM INTO`, `pg_dump -F c`, téléchargement + `PRAGMA quick_check`/`pg_restore`) et `RetentionManager` (`MONITOR_RETENTION_DAYS=90`).

## 5. Observabilité Prometheus, Alertes & Wails `--demo`
* **Décision** : Ajout d'`internal/metrics` (`/metrics`), liaison directe `alert_dispatch_logs.telemetry_id`, retries backoff (`500ms`/`1500ms`) + `AlertService.Shutdown(ctx)` et mode `--demo`.
