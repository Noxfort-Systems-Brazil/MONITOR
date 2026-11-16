# 📡 Référence API et Protocoles d'Ingestion

Ce document spécifie formellement les interfaces de communication de **Noxfort Monitor™ v2.0** : ingestion asynchrone MQTT, appairage et télémétrie HTTP Zero-Trust, endpoints REST d'administration sous RBAC et observabilité Prometheus.

⬅️ [Hub Central](../README.md) | 🏛️ [Architecture](architecture.md) | 🗄️ [Base de Données](database.md) | 🔐 [Sécurité](security.md)

---

## 1. Protocole d'Ingestion MQTT

* **Adresse du Broker** : `tcp://127.0.0.1:1883` (configurable dans `/settings` avec `Reconnect(brokerURL)` à chaud et boucle de reprise de 5s au démarrage)
* **Modèle de Topic par Défaut** : `noxfort/telemetry/{identifier}`
* **Souscription Wildcard** : `noxfort/telemetry/#`
* **Politique Stricte (`MONITOR_MQTT_STRICT`)** : Activée par défaut (`true`) ; rejette les messages issus d'équipements non enregistrés ou en état `PAUSED`.

### Format JSON Universel `IncomingEvent`

```json
{
  "category": "HARDWARE",
  "origin": "sensor-presion-01",
  "level": "CRITICAL",
  "message": "Surchauffe détectée : Température 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

#### Spécification des Champs :
* **`category`** (*String*) : `HARDWARE` ou `SOFTWARE` (route les alertes par rôle : Techniciens / Développeurs).
* **`origin`** (*String*) : Identifiant unique de l'équipement (ex : `carina`, `synapse`, `pump-01`).
* **`level`** (*String*) : `INFO`, `WARNING` ou `CRITICAL` (les messages `INFO` heartbeat actualisent `last_seen` sans bruit d'alerte ; `CRITICAL` déclenche `SaveEventReturningID` et les alertes liées à `telemetry_id`).
* **`message`** (*String*) : Description lisible de l'événement.
* **`occurred_at`** (*String ISO-8601*) : Horodatage UTC (validé dans une fenêtre de `±300s` en HTTP).

---

## 2. Appairage et Ingestion HTTP Zero-Trust

### 2.1 Pré-enregistrement : `POST /api/pair`
* **Port** : `22100` (ou URL publique HTTPS via Caddy/Ngrok)
* **Authentification** : Publique (corps limité à `64 Ko`, file limitée à `50` équipements `PENDING` $\rightarrow$ `429 Too Many Requests`).
* **Corps JSON** : `{"origin": "synapse-core", "name": "Synapse Core"}`
* **Réponses** :
  * `202 Accepted` : Placé en file d'attente `PENDING`. **Aucune clé API n'est renvoyée** dans cette réponse (un `ADMIN` valide et copie la clé temporaire `nox_live_` à TTL de 5 min hors bande depuis `/devices`).
  * `409 Conflict` : L'équipement est déjà enregistré comme `ACTIVE`.

### 2.2 Ingestion Télémétrique : `POST /api/telemetry`
* **Authentification** : Obligatoire par en-tête `X-Device-Token: <api_key>` ou `Authorization: Bearer <api_key>` (comparaison en temps constant).
* **Anti-Spoofing et Anti-Rejeu** : Vérifie `X-Device-Origin` contre l'origine enregistrée et rejette tout `occurred_at` hors de la fenêtre `±300s`.
* **Réponses** : `202 Accepted` (`BufferedTelemetryWriter`), `401 Unauthorized` (jeton manquant/invalide/expiré) ou `403 Forbidden` (équipement `PAUSED` ou divergence d'origine).

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_votre_jeton" \
  -H "X-Device-Origin: synapse-core" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Pool de connexions à 85% de capacité.",
    "occurred_at": "2026-09-27T18:00:00Z"
  }'
```

---

## 3. Endpoints d'Administration et d'Exploitation

Toutes les routes protégées utilisent des cookies de session (`DBSessionStore`), rejettent `?token=` en paramètre URL et appliquent le RBAC (`OPERATOR` strictement en lecture seule + changement de son propre mot de passe).

| Endpoint | Méthode | Rôle Minimum | Description |
|---|---|---|---|
| `/api/auth/login` | `POST` | Public | Authentifie l'utilisateur ; retourne `429` après 5 échecs en 15m (verrouillage 5m). |
| `/api/auth/logout` | `POST` | Authentifié | Révoque l'empreinte de session dans `sessions`. |
| `/api/auth/status` | `GET` | Public | Retourne l'état d'authentification, l'utilisateur et son rôle. |
| `/api/auth/register` | `POST` | Admin | Création administrative soumise à `ValidatePasswordComplexity`. |
| `/api/dashboard/summary` | `GET` | Opérateur+ | Résumé JSON temps réel (polling 10s) sur 5 états : `ONLINE`, `OFFLINE`, `PENDING`, `PAUSED`, `WAITING`. |
| `/devices` | `GET` | Opérateur+ | Liste les équipements (secrets `ApiKey` actifs masqués dans le HTML). |
| `/devices/create` / `update` / `toggle` | `POST` | Admin | Provisionne (clé à usage unique), renomme ou bascule en Maintenance (`PAUSED`). |
| `/devices/rotate-key` / `delete` | `POST` | Admin | Renouvelle la clé API ou supprime définitivement l'équipement. |
| `/contacts` et `/contacts/{create,update,delete}` | `GET`/`POST` | Op+ / Admin | Consultation (`Operator+`) et gestion (`Admin`) des destinataires d'alerte. |
| `/api/users` et `/api/users/{create,delete}` | `GET`/`POST` | Admin | Gestion des comptes (protège le superutilisateur et le dernier admin). |
| `/api/users/password` | `POST` | Opérateur+ | Modification du propre mot de passe (exige `current_password`). |
| `/api/users/role` et `/api/users/unlock` | `POST` | Admin | Met à jour le rôle RBAC ou déverrouille un compte bloqué par `LoginAttemptTracker`. |
| `/api/settings/database/status` / `test` / `save` / `provision-user` | `GET`/`POST` | Op+ / Admin | Diagnostic et basculement SQLite $\leftrightarrow$ PostgreSQL via `MigrateData` transactionnel. |
| `/api/settings/database/backups` / `backup` | `GET`/`POST` | Admin | Liste ou déclenche un snapshot à chaud (`VACUUM INTO` / `pg_dump -F c`). |
| `/api/settings/database/backups/download` / `restore` | `GET`/`POST` | Admin | Téléchargement sécurisé (`ResolveBackupPath`) et restauration (`PRAGMA quick_check` / `pg_restore`). |
| `/api/tunnel/{status,save,start,stop,disconnect,test}` | `GET`/`POST` | Op+ / Admin | Lecture (`Operator+`) et pilotage (`Admin`) DuckDNS / Ngrok. |
| `/api/audit/security` / `alerts` / `transitions` | `GET` | Opérateur+ | Consulte les journaux de sécurité, d'alertes et de transitions d'état. |
| `/settings/test` et `/settings/test-telegram` | `POST` | Admin | Envoie un e-mail de test (TLS implicite 465 / CRLF filtré) ou une alerte Telegram. |

---

## 4. Endpoints Publics d'Observabilité et de Diagnostic

| Endpoint | Méthode | Description | Format |
|---|---|---|---|
| `/healthz` | `GET` | Sonde de vivacité (répond `200 OK`). | `text/plain` |
| `/api/health` | `GET` | Diagnostic complet : état/latence BD, état MQTT, mémoire, goroutines et équipements. | `application/json` |
| `/metrics` | `GET` | Métriques **Prometheus** : `noxfort_uptime_seconds`, `noxfort_goroutines`, `noxfort_memory_heap_alloc_bytes`, `noxfort_memory_heap_sys_bytes`, `noxfort_memory_num_gc`, `noxfort_db_status`, `noxfort_db_latency_ms`, `noxfort_mqtt_connected`, `noxfort_devices_total`, `noxfort_devices_by_status`, `noxfort_telemetry_buffer_depth`, `noxfort_telemetry_ingested_total`, `noxfort_telemetry_rejected_total`, `noxfort_alerts_dispatched_total`. | `text/plain; version=0.0.4` |
