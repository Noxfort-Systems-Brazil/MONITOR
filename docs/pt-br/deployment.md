# 🚀 Guia de Deploy em Produção

Este documento fornece instruções passo a passo para a implantação do **Noxfort Monitor™ v2.0** em ambientes corporativos e industriais, cobrindo o serviço Systemd headless, proxy reverso NGINX com terminação SSL e instalação via pacote Debian (`.deb`).

⬅️ [Central de Documentação](../README.md) | 🏛️ [Arquitetura](architecture.md) | 🖥️ [App Desktop](desktop_app.md)

---

## 1. Instalação via Pacote Debian (`.deb`)

```bash
# Compilar o instalador .deb
make deb

# Instalar no servidor Ubuntu/Debian
sudo dpkg -i build_deb/noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f
```

Instala os binários em `/opt/noxfort-monitor/`, cria o link simbólico `/usr/local/bin/noxfort-monitor`, registra os atalhos de sistema e habilita o serviço Mosquitto MQTT.

---

## 2. Serviço Systemd Dedicado (Modo Headless)

Recomendado para servidores em nuvem e instâncias sem monitor:

### 2.1 Arquivo de Unidade (`/etc/systemd/system/noxfort-monitor.service`)
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

### 2.2 Ativação e Monitoramento
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now noxfort-monitor
sudo journalctl -u noxfort-monitor -f
```

---

## 3. Proxy Reverso e Gateway HTTPS (Caddy ou NGINX)

### 3.1 Opção A: Caddy Server com DuckDNS (Recomendado)
O ecossistema disponibiliza gateway Caddy em contêiner com validação automática de certificados via **DNS-01 ACME**:
```bash
make caddy-build
make services-start
```
Consulte o guia completo em [Integração Caddy & DuckDNS](../CADDY_INTEGRATION.md).

### 3.2 Opção B: NGINX para Ingestão de Telemetria
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

    # Encaminha telemetria de sensores e agentes IoT
    location /api/telemetry {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Acessos web ao dashboard retornam 403 Forbidden por design de segurança
    location / {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
> [!NOTE]
> O acesso de navegadores ao painel web está desativado no servidor HTTP (retornando 403 Forbidden). A operação da interface ocorre exclusivamente no aplicativo desktop nativo.
