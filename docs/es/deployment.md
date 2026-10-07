# 🚀 Guía de Despliegue en Producción

Este documento proporciona las instrucciones paso a paso para desplegar **Noxfort Monitor™ v2.0** en entornos industriales y corporativos, cubriendo el servicio systemd headless, proxy inverso NGINX con terminación SSL e instalación con paquete Debian (`.deb`).

⬅️ [Hub Central](../README.md) | 🏛️ [Arquitectura](architecture.md) | 🖥️ [App Desktop](desktop_app.md)

---

## 1. Instalación mediante Paquete Debian (`.deb`)

```bash
# Compilar el paquete instalador .deb (generado en la raíz como noxfort-monitor_2.0.1_amd64.deb)
make deb

# Opción A: Instalación o actualización automática vía Makefile
make install-deb

# Opción B: Instalación directa vía dpkg
sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f

# Para desinstalar:
make uninstall-deb
```

Instala los binários en `/opt/noxfort-monitor/`, genera el enlace simbólico `/usr/local/bin/noxfort-monitor`, añade accesos directos y habilita el servicio Mosquitto MQTT.

---

## 2. Servicio Systemd Dedicado (Modo Headless)

Recomendado para servidores en la nube y máquinas sin pantalla:

### 2.1 Archivo de Servicio (`/etc/systemd/system/noxfort-monitor.service`)
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

### 2.2 Habilitación e Inicio
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now noxfort-monitor
sudo journalctl -u noxfort-monitor -f
```

---

## 3. Proxy Inverso y Pasarela HTTPS (Caddy o NGINX)

### 3.1 Opción A: Servidor Caddy con DuckDNS (Recomendada)
El ecosistema incluye una pasarela Caddy en contenedor con emisión automática de certificados mediante **DNS-01 ACME**:
```bash
make caddy-build
make services-start
```
Consulte la guía completa en [Integración Caddy y DuckDNS](caddy_integration.md).

### 3.2 Opción B: NGINX para Ingestión con SSL

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
