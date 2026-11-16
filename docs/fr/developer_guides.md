# 👨‍💻 Guide Développeur Avancé : Noxfort Monitor™

Ce document fournit les directives d'ingénierie logicielle pour compiler, maintenir et étendre **Noxfort Monitor™ v2.0** : `Makefile`, contrat `ReloadableRepository`, utilitaires HTTP de sécurité, métriques Prometheus et gestion de cycle de vie.

⬅️ [Centre de Documentation](README.md) | 🏛️ [Architecture](architecture.md) | 🧪 [Tests & QA](testing.md) | 🗄️ [Base de Données](database.md)

---

## 1. Environnement Local & Commandes `Makefile`

| Commande | Description |
| :--- | :--- |
| `make build` | Compile le binaire de bureau (`bin/noxfort-monitor`). |
| `make run` | Lance l'application en développement avec UI Wails v2. |
| `make run-headless` | Démarre le serveur en mode `--headless`. |
| `make test` | Exécute les tests Go (`go test ./... -v`) et frontend Vitest (`npm test`). |
| `make test-frontend` | Exécute uniquement la suite Vitest (`npm test`). |
| `make lint` | Lance la vérification statique (`go vet`). |
| `make record-demo` | Exécute les scripts `scripts/demo/` contre `--demo`. |
| `make deb` / `make install-deb` / `make uninstall-deb` | Gère le paquet Debian `.deb`. |
| `make broker-start` / `make broker-stop` | Démarre ou arrête Mosquitto local. |

---

## 2. Contrats de Conception, Persistance & Sécurité

* **Racine de Composition** : Instanciation explicite dans [`cmd/server/main.go`](../../cmd/server/main.go) avec chargement de `.env` via `LoadDotEnv()`.
* **Contrat `ReloadableRepository` (`7` dépôts)** : `Device`, `Telemetry`, `Settings`, `Contact`, `Audit`, `User` et `DBSessionStore` implémentent `SetDB(db *sql.DB, driver string)` pour supporter la migration transactionnelle (`MigrateData` sur 9 tables) SQLite WAL $\leftrightarrow$ PostgreSQL à chaud.
* **Conventions HTTP & Sécurité** :
  * Utilisez `RenderPageWithRequest` et `writeForbiddenResponse` (JSON `403` ou [`web/templates/forbidden.html`](../../web/templates/forbidden.html)).
  * Utilisez `ExtractSessionToken` (rejette `?token=`), `ExtractClientIP` et `IsHTTPSRequest` (`TRUSTED_PROXIES`, `MONITOR_USE_HTTPS`).
  * Masquez toujours `ApiKey` et `duckdns_token` dans les vues HTML.
* **Métriques, Alertes & Extinction (`bgCtx`)** :
  * Instrumentez `internal/metrics` (`IncTelemetryIngested`, `IncTelemetryRejected`, `IncAlertDispatched`, `SetBufferDepth`).
  * Passez par `BufferedTelemetryWriter` pour la télémétrie courante et `SaveEventReturningID` + `TriggerAlertWithTelemetryID` pour lier `alert_dispatch_logs.telemetry_id` ; drainez avec `AlertService.Shutdown(shutdownCtx)`.
