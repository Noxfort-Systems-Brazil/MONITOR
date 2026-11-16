# 🚀 Guía de Despliegue en Producción

Este documento proporciona las instrucciones actualizadas para desplegar **Noxfort Monitor™ v2.0** mediante variables `.env` (tras la eliminación de `configs/config.yaml`), paquete Debian (`.deb`), servicio systemd headless, Docker Compose, proxy inverso, backups programados y Prometheus.

⬅️ [Hub Central](../README.md) | 🏛️ [Arquitectura](architecture.md) | 🖥️ [App Desktop](desktop_app.md)

---

## 1. Configuración por Entorno (`.env`)

`configs/config.yaml` ha sido eliminado. Los parámetros de arranque se cargan desde `.env` mediante `LoadDotEnv()` (consulte `.env.example`), mientras las configuraciones dinámicas (URL del broker MQTT, alertas, túneis) residen en SQL (`settings`).

| Variable | Por Defecto | Descripción |
|---|---|---|
| `MONITOR_MQTT_STRICT` | `true` | Descarta telemetría MQTT (`noxfort/telemetry/#`) de nodos no registrados o `PAUSED`. |
| `TRUSTED_PROXIES` | *(vacío)* | CIDRs/IPs de proxies autorizados para confiar en `X-Real-IP`, `X-Forwarded-For` y `X-Forwarded-Proto`. |
| `MONITOR_USE_HTTPS` | `false` | Fuerza cookies `Secure` y detección HTTPS detrás de proxies TLS. |
| `MONITOR_RETENTION_DAYS` | `90` | Días de retención antes de que `RetentionManager` purgue `telemetry` y logs (cada 24h). |
| `MONITOR_BACKUP_INTERVAL_HOURS` | `24` | Cadencia de snapshots automáticos de `BackupManager` (conserva hasta `7` copias). |
| `MONITOR_DB_DRIVER` / `MONITOR_DB_*` | `sqlite` | Parámetros opcionales de arranque directo con PostgreSQL. |

---

## 2. Paquete Debian (`.deb`) y Docker Compose

```bash
# Paquete .deb
make deb
make install-deb        # O: sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb && sudo apt-get install -f
make uninstall-deb

# Docker Compose (--headless + Mosquitto + PostgreSQL opcional)
cp .env.example .env
docker compose up -d --build
./scripts/setup_mqtt_auth.sh
```

---

## 3. Servicio Systemd Dedicado (Modo Headless)

```ini
[Unit]
Description=Noxfort Monitor Server (Modo Headless)
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

## 4. Proxy Inverso (Caddy o NGINX), Backups y `/metrics`

* **Opción A — Caddy (`caddy/Caddyfile`)**: `make caddy-build && make services-start` (consulte [Caddy y DuckDNS](caddy_integration.md)).
* **Opción B — NGINX**: Agregue la IP de NGINX a `TRUSTED_PROXIES` y reenvíe las cabeceras estándar:

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

* **Backups, Retención y Prometheus**: `BackupManager` programa snapshots `VACUUM INTO` / `pg_dump -F c` con soporte de descarga/restauración (`/api/settings/database/restore`), `RetentionManager` purga registros antiguos cada 24h y `GET /metrics` expone métricas Prometheus para recolección.
