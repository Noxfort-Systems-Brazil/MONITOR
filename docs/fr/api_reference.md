# 📡 Référence API et Protocoles d'Ingestion

Ce document spécifie formellement les interfaces réseau de **Noxfort Monitor™ v2.0**, couvrant l'ingestion MQTT asynchrone, l'endpoint HTTP REST de télémétrie et les API d'administration.

⬅️ [Hub Central](../README.md) | 🏛️ [Architecture](architecture.md) | 🗄️ [Base de Données](database.md) | 🔐 [Sécurité](security.md)

---

## 1. Protocole d'Ingestion MQTT

* **Adresse du Broker** : `tcp://127.0.0.1:1883`
* **Topic par Défaut** : `noxfort/devices/{identifier}/telemetry`
* **Abonnement Générique** : `noxfort/devices/+/telemetry`

### Schéma JSON Universel `IncomingEvent`

```json
{
  "category": "HARDWARE",
  "origin": "capteur-pression-01",
  "level": "CRITICAL",
  "message": "Surchauffe détectée : Température 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

#### Définition des Champs :
* **`category`** (*String*) : `HARDWARE` ou `SOFTWARE`. Détermine les règles de routage RBAC (Matériel $\rightarrow$ Techniciens ; Logiciel $\rightarrow$ Programmeurs).
* **`origin`** (*String*) : Identifiant unique de l'appareil (ex : `carina`, `synapse`, `pump-01`).
* **`level`** (*String*) : `INFO`, `WARNING` ou `CRITICAL`.
  * Les messages `INFO` contenant des mots-clés de maintien en vie ("*system ok*", "*heartbeat*", "*online*") mettent à jour l'horodatage de présence sans générer d'alertes ni d'écritures superflues en base de données.
  * Les messages `CRITICAL` déclenchent l'envoi immédiat d'alertes multicanal.
* **`message`** (*String*) : Description textuelle de l'événement.
* **`occurred_at`** (*String ISO-8601*) : Horodatage UTC.

---

## 2. Ingestion Télémétrique via HTTP REST

### `POST /api/telemetry`
Conçu pour les nœuds distants (**Carina**, **Synapse**, microcontrôleurs IoT ou scripts cURL).

* **Port** : `22100` (ou URL publique DuckDNS/Caddy/Ngrok)
* **Authentification** : Publique (dispensée de middleware pour accueillir les capteurs autonomes).
* **Isolation Réseau** : En production (`ExternalIngestionHandler`), `POST /api/telemetry`, `/healthz`, `/api/health` et `/metrics` sont accessibles sur le port `22100`. Les requêtes directes de navigateurs vers le tableau de bord renvoient **403 Forbidden**.
* **En-tête** : `Content-Type: application/json`
* **Corps** : Identique au schéma JSON `IncomingEvent`.

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Pool de connexions saturé à 85%.",
    "occurred_at": "2026-09-27T18:00:00Z"
  }'
```

---

## 3. Endpoints d'Administration et d'Exploitation

| Endpoint | Méthode | Rôle Requis | Description |
|---|---|---|---|
| `/api/auth/login` | `POST` | Public | Valide les identifiants et génère le cookie `noxfort_session`. |
| `/api/auth/logout` | `POST` | Authentifié | Révoque le jeton de session actif. |
| `/api/auth/status` | `GET` | Public | Retourne l'état de session, nom d'utilisateur et rôle. |
| `/api/auth/register` | `POST` | Admin | Alias protégé de création de compte avec validation de complexité du mot de passe. |
| `/devices` | `GET` | Opérateur+ | Affiche l'écran de gestion des équipements avec `LastSeen`. |
| `/devices/delete` | `POST` | Admin | Supprime un équipement via formulaire POST (`id=xyz`) et redirige (`303`). |
| `/contacts` | `GET` | Opérateur+ | Affiche l'interface de gestion de l'Équipe d'Intervention. |
| `/contacts/create` | `POST` | Admin | Enregistre un nouveau contact d'alerte (nom, email, téléphone, telegram_chat_id, rôle). |
| `/contacts/update` | `POST` | Admin | Met à jour les coordonnées et canaux d'un contact existant. |
| `/contacts/delete` | `POST` | Admin | Supprime un contact de l'équipe d'intervention. |
| `/api/users` | `GET` | Admin | Retourne la liste JSON de tous les opérateurs enregistrés. |
| `/api/users/create` | `POST` | Admin | Crée un compte opérateur ou administrateur (mot de passe 8+ caractères requis). |
| `/api/users/delete` | `POST` | Admin | Supprime un compte utilisateur (le superutilisateur est protégé). |
| `/api/settings/database/status` | `GET` | Authentifié | Retourne le moteur actif (Postgres/SQLite), version et latence. |
| `/api/settings/database/test` | `POST` | Admin | Teste la connectivité PostgreSQL avant basculement. |
| `/api/settings/database/save` | `POST` | Admin | Recharge à chaud les dépôts actifs avec le moteur cible (`migrate=true` supporté). |
| `/api/settings/database/provision-user` | `POST` | Admin | Crée un utilisateur dédié avec droits restreints sur le schéma PostgreSQL. |
| `/api/settings/database/backup` | `POST` | Admin | Déclenche un instantané à chaud cohérent de la base (`VACUUM INTO` / `pg_dump`). |
| `/api/settings/database/backups` | `GET` | Admin | Liste les instantanés de sauvegarde disponibles dans `backups/`. |
| `/api/tunnel/status` | `GET` | Authentifié | Retourne l'état WAN, fournisseur (DuckDNS/Ngrok), URLs et IPv6. |
| `/api/tunnel/save` | `POST` | Admin | Enregistre les paramètres DuckDNS / Ngrok dans la base de données. |
| `/api/tunnel/start` | `POST` | Admin | Déclenche la mise à jour dynamique DuckDNS ou ouvre le tunnel. |
| `/api/tunnel/stop` | `POST` | Admin | Interrompt les mises à jour et désactive le démarrage automatique. |
| `/api/tunnel/disconnect` | `POST` | Admin | Supprime les identifiants enregistrés et arrête le service. |
| `/api/tunnel/test` | `POST` | Admin | Valide activement le jeton et la résolution DNS de DuckDNS. |
| `/api/audit/security` | `GET` | Admin | Consulte les journaux d'accès et d'audit de sécurité. |
| `/api/audit/alerts` | `GET` | Admin | Consulte les journaux d'expédition et SLA d'alertes. |
| `/api/audit/transitions` | `GET` | Admin | Consulte l'historique des pannes et durées d'indisponibilité. |
| `/settings/test` | `POST` | Admin | Envoie un email de test avec la configuration SMTP active. |
| `/settings/test-telegram` | `POST` | Admin | Envoie une alerte de test MarkdownV2 vers Telegram. |
| `/api/open-external` | `POST` | Authentifié | Ouvre une URL dans le navigateur par défaut du système. |
| `/api/window/toggle-fullscreen` | `POST` | Public | Bascule la fenêtre Wails en plein écran. |
| `/api/window/exit-fullscreen` | `POST` | Public | Quitte le mode plein écran. |

---

## 4. Endpoints Publics d'Observabilité et Diagnostics

*Exempts d'authentification pour scraping Prometheus et sondes de conteneurs.*

| Endpoint | Méthode | Description | Format |
|---|---|---|---|
| `/healthz` | `GET` | Sonde de vivacité liveness (réponse 200 `OK`). | `text/plain` |
| `/api/health` | `GET` | Diagnostic approfondi : santé et latence BD, statut MQTT, mémoire, goroutines et nœuds. | `application/json` |
| `/metrics` | `GET` | Métriques au format **Prometheus** (`noxfort_uptime_seconds`, `noxfort_goroutines`, etc.). | `text/plain; version=0.0.4` |
