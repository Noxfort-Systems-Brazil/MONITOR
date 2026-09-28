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

* **Port** : `22100` (ou URL publique Ngrok)
* **Authentification** : Publique (dispensée de middleware pour accueillir les capteurs autonomes).
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
| `/api/devices` | `GET` | Opérateur+ | Liste les équipements enregistrés et le statut `LastSeen`. |
| `/api/devices/register` | `POST` | Admin | Enregistre ou met à jour un équipement. |
| `/api/settings/database/test` | `POST` | Admin | Teste la connectivité PostgreSQL avant basculement. |
| `/api/settings/database/migrate`| `POST` | Admin | Migre les tables de données entre SQLite et PostgreSQL. |
| `/api/settings/database/save` | `POST` | Admin | Recharge à chaud les dépôts actifs avec le moteur cible. |
| `/api/tunnel/start` | `POST` | Admin | Démarre le tunnel inverse Ngrok. |
| `/api/tunnel/stop` | `POST` | Admin | Ferme le tunnel inverse. |
| `/api/audit/security` | `GET` | Admin | Consulte les journaux d'accès et d'audit de sécurité. |
| `/api/audit/alerts` | `GET` | Admin | Consulte les journaux d'expédition et SLA d'alertes. |
