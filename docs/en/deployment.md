# 🚀 Production Deployment Guide

This document provides deployment instructions for **Noxfort Monitor™ v2.0**, covering systemd service installation in headless mode, NGINX SSL reverse proxy setup, and Debian package installation.

⬅️ [Central Hub](../README.md) | 🏛️ [Architecture](architecture.md) | 🖥️ [Desktop App](desktop_app.md)

---

## 1. Debian Package Installation (`.deb`)

```bash
# Build the .deb installer
make deb

# Install on target Ubuntu/Debian host
sudo dpkg -i build_deb/noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f
```

Installs the executable to `/opt/noxfort-monitor/`, creates `/usr/local/bin/noxfort-monitor`, registers system shortcuts, and enables Mosquitto MQTT.

---

## 2. Dedicated Headless Systemd Service

For cloud VPS and headless servers:

### 2.1 Service File (`/etc/systemd/system/noxfort-monitor.service`)
```ini
[Unit]
Description=Noxfort Monitor Server (Headless Mode)
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

### 2.2 Enabling and Starting
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now noxfort-monitor
sudo journalctl -u noxfort-monitor -f
```

---

## 3. Reverse Proxy & HTTPS Gateway (Caddy or NGINX)

### 3.1 Option A: Caddy Server with DuckDNS (Recommended)
The ecosystem includes a containerized Caddy edge gateway with automated **DNS-01 ACME** certificate provisioning:
```bash
make caddy-build
make services-start
```
See the complete guide in [Caddy & DuckDNS Integration](../CADDY_INTEGRATION.md).

### 3.2 Option B: NGINX for Ingestion Proxy
```nginx
server {
    listen 80;
    server_name monitor.noxfort.internal;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name monitor.noxfort.internal;

    ssl_certificate /etc/letsencrypt/live/monitor.noxfort.internal/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/monitor.noxfort.internal/privkey.pem;

    # Forwards telemetry from remote edge nodes
    location /api/telemetry {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Browser requests to root return 403 Forbidden by design
    location / {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
> [!NOTE]
> Web browser access to the dashboard is deactivated on external HTTP ports (returning 403 Forbidden). GUI management operations take place exclusively in the native desktop application.
