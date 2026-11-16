# 🚀 Guide de Déploiement en Production

Ce document décrit le déploiement en production de **Noxfort Monitor™ v2.0** via `.env` (après suppression de `configs/config.yaml`), paquet Debian (`.deb`), systemd headless, Docker Compose, proxy inverse, sauvegardes automatisées et Prometheus.

⬅️ [Hub Central](../README.md) | 🏛️ [Architecture](architecture.md) | 🖥️ [Application Desktop](desktop_app.md)

---

## 1. Configuration d'Environnement (`.env`)

`configs/config.yaml` a été supprimé. Les paramètres de démarrage chargent depuis `.env` via `LoadDotEnv()` (`.env.example`), tandis que la configuration dynamique (URL MQTT, alertes, tunnels) est stockée dans la table SQL `settings`.

| Variable | Défaut | Description |
|---|---|---|
| `MONITOR_MQTT_STRICT` | `true` | Ignore les publications MQTT (`noxfort/telemetry/#`) d'équipements inconnus ou `PAUSED`. |
| `TRUSTED_PROXIES` | *(vide)* | CIDRs/IPs autorisés pour `X-Real-IP`, `X-Forwarded-For` et `X-Forwarded-Proto`. |
| `MONITOR_USE_HTTPS` | `false` | Active les cookies `Secure` et la détection HTTPS derrière TLS. |
| `MONITOR_RETENTION_DAYS` | `90` | Durée avant purge automatique (`RetentionManager`) de `telemetry`, audits, alertes et transitions. |
| `MONITOR_BACKUP_INTERVAL_HOURS` | `24` | Fréquence des snapshots à chaud (`BackupManager`, conserve `7` archives). |
| `MONITOR_DB_DRIVER` / `MONITOR_DB_*` | `sqlite` | Configuration optionnelle pour démarrage direct sur PostgreSQL. |

---

## 2. Paquet Debian (`.deb`) et Docker Compose

```bash
# Paquet Debian
make deb
make install-deb        # Ou : sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb && sudo apt-get install -f
make uninstall-deb

# Docker Compose (--headless + Mosquitto + PostgreSQL optionnel)
cp .env.example .env
docker compose up -d --build
./scripts/setup_mqtt_auth.sh
```

---

## 3. Service Systemd (Mode Headless)

```ini
[Unit]
Description=Noxfort Monitor Server (Mode Headless)
After=network.target postgresql.service mosquitto.service

[Service]
Type=simple
User=noxfort
Group=noxfort
ExecStart=/usr/local/bin/noxfort-monitor --headless
WorkingDirectory=/home/noxfort

Environment="PORT=22100"
EnvironmentFile=-/home/noxfort/.env

Restart=always
RestartSec=5s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

---

## 4. Proxy Inverse (Caddy / NGINX), Sauvegardes et `/metrics`

* **Option A — Caddy (`caddy/Caddyfile`)** : `make caddy-build && make services-start` ([Intégration Caddy](caddy_integration.md)).
* **Option B — NGINX** : Déclarez l'IP NGINX dans `TRUSTED_PROXIES` et transmettez :

```nginx
location / {
    proxy_pass http://127.0.0.1:22100;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

* **Sauvegardes, Rétention et Prometheus** : `BackupManager` automatise `VACUUM INTO` / `pg_dump -F c` avec téléchargement et restauration (`/api/settings/database/restore`), `RetentionManager` purge les logs toutes les 24h et `GET /metrics` expose les métriques Prometheus.
