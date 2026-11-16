# 🧪 Tests et Assurance Qualité (QA)

Ce document décrit les procédures de validation de **Noxfort Monitor™ v2.0** : suites automatisées Go + Vitest, tests E2E via MQTT (`noxfort/telemetry/#`) et HTTP Zero-Trust (`/api/pair` + `/api/telemetry`), et mode démo automatisé.

⬅️ [Hub Central](../README.md) | 🏛️ [Architecture](architecture.md) | 📡 [API](api_reference.md)

---

## 1. Suites de Tests Automatisées

```bash
# Exécute le backend Go (go test ./... -v) et les tests frontend Vitest (npm test)
make test

# Tests frontend uniquement (Vitest : 4 fichiers, 26 tests ; remplace l'ancien usage de pytest)
make test-frontend

# Analyse statique Go (go vet)
make lint
```

* **Backend Go** : Couvre sécurité (`lockout_test.go`, hachage PBKDF2, complexité), stockage (`session_repo_test.go`, `retention_test.go`, migration transactionnelle, sauvegarde/restauration), middleware RBAC HTTP, MQTT strict, décodeur de protocole et alertes (`alerts_test.go`).
* **Mocks en Mémoire** : `MockDeviceRepository`, `MockTelemetryRepository`, `MockAlertDispatcher` et `MockNotificationChannel`.

---

## 2. Vérification Manuelle E2E

### 2.1 Ingestion MQTT (`mosquitto_pub`)
Avec `MONITOR_MQTT_STRICT=true` (par défaut), enregistrez `pompe-01` comme `ACTIVE` dans `/devices` (ou définissez `MONITOR_MQTT_STRICT=false` en dev local) avant de publier sur `noxfort/telemetry/#` :

```bash
# Battement de cœur
mosquitto_pub -t "noxfort/telemetry/pompe-01" -m '{
  "category": "HARDWARE",
  "origin": "pompe-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# Alarme critique (liée dans alert_dispatch_logs via telemetry_id)
mosquitto_pub -t "noxfort/telemetry/pompe-01" -m '{
  "category": "HARDWARE",
  "origin": "pompe-01",
  "level": "CRITICAL",
  "message": "Alarme de Surpression Hydraulique : 180 bar",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 Ingestion HTTP Zero-Trust (`cURL`)

```bash
# 1. Pré-enregistrement Zero-Trust (202 Accepted sans clé API)
curl -i -X POST http://localhost:22100/api/pair \
  -H "Content-Type: application/json" \
  -d '{"origin":"carina-core","name":"Carina Core"}'

# 2. Approuver dans /devices, copier le jeton nox_live_ (TTL 5m), puis envoyer :
curl -i -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_VOTRE_JETON_COPIE" \
  -H "X-Device-Origin: carina-core" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "Latence d’inférence 4.2ms (seuil : 3.5ms)",
    "occurred_at": "'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"
  }'
```
*Réponse attendue* : `202 Accepted`, mise à jour dans `GET /api/dashboard/summary` et incrémentation sur `GET /metrics`.

---

## 3. Mode Démo Automatisé
Exécutez `make record-demo` (`scripts/demo/record_presentation.cjs` et `mock_telemetry.cjs`) sur `--demo` (`RunDesktopServer()`).
