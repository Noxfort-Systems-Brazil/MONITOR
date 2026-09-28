# 🚀 Guide de Déploiement en Production

Ce document fournit les instructions de déploiement de **Noxfort Monitor™ v2.0**, couvrant l'installation du service systemd en mode headless, la configuration d'un proxy inverse NGINX avec terminaison SSL et l'installation via paquet Debian (`.deb`).

⬅️ [Hub Central](../README.md) | 🏛️ [Architecture](architecture.md) | 🖥️ [Application Desktop](desktop_app.md)

---

## 1. Installation par Paquet Debian (`.deb`)

```bash
# Génération du paquet .deb
make deb

# Installation sur l'hôte Ubuntu/Debian
sudo dpkg -i build_deb/noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f
```

Le paquet installe l'exécutable dans `/opt/noxfort-monitor/`, crée le lien `/usr/local/bin/noxfort-monitor`, configure les raccourcis de bureau et active le service Mosquitto MQTT.

---

## 2. Service Systemd Dédié (Mode Headless)

Recommandé pour les serveurs en nuage et machines sans écran :

### 2.1 Fichier d'Unité (`/etc/systemd/system/noxfort-monitor.service`)
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

### 2.2 Activation et Surveillance
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now noxfort-monitor
sudo journalctl -u noxfort-monitor -f
```

---

## 3. Proxy Inverse NGINX avec SSL

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
}
```
