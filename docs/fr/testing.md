# 🧪 Tests et Assurance Qualité (QA)

Ce document décrit les procédures de test de **Noxfort Monitor™ v2.0**, comprenant les tests unitaires automatisés, les mocks de dépôts en mémoire et la vérification manuelle E2E par MQTT et HTTP REST.

⬅️ [Hub Central](../README.md) | 🏛️ [Architecture](architecture.md) | 📡 [API](api_reference.md)

---

## 1. Suite de Tests Automatisés

Lancement de la suite complète de tests :
```bash
make test
# Équivalent à : go test ./... -v
```

### Architecture des Mocks en Mémoire
La logique métier dans `internal/monitor` et `internal/security` s'affranchit des bases de données sur disque grâce aux interfaces de simulation :
* `MockDeviceRepository` : Simule la consultation des nœuds et la mise à jour de `LastSeen`.
* `MockTelemetryRepository` : Vérifie la persistance des incidents.
* `MockAlertDispatcher` : Confirme la transmission des alertes vers la politique de routage.
* `MockNotificationChannel` : Teste l'envoi d'alertes sans solliciter de serveurs SMTP ou Telegram réels.

---

## 2. Vérification Manuelle de Bout en Bout (E2E)

### 2.1 Ingestion MQTT (`mosquitto_pub`)

```bash
# Battement de cœur (Keep-Alive)
mosquitto_pub -t "noxfort/devices/pompe-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pompe-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# Alarme critique (Déclenche le routage d'alertes)
mosquitto_pub -t "noxfort/devices/pompe-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pompe-01",
  "level": "CRITICAL",
  "message": "Alarme de Surpression Hydraulique : 180 bar",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 Ingestion HTTP REST (`cURL`)

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "Latence d’inférence 4.2ms (seuil : 3.5ms)",
    "occurred_at": "2026-09-27T18:10:00Z"
  }'
```
*Réponse attendue* : Code HTTP `200 OK` avec `{"status":"received"}`.
