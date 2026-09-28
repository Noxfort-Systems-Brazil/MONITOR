# 🚀 生产环境部署指南

本文档提供在工业级与企业级生产环境中部署 **Noxfort Monitor™ v2.0** 的详细步骤，包括基于 Systemd 的无头服务安装、NGINX SSL 反向代理配置及 Debian 软件包 (`.deb`) 安装方式。

⬅️ [文档中心](../README.md) | 🏛️ [系统架构](architecture.md) | 🖥️ [桌面应用](desktop_app.md)

---

## 1. 通过 Debian 软件包安装 (`.deb`)

```bash
# 构建 .deb 安装包
make deb

# 在目标 Ubuntu/Debian 机器上安装
sudo dpkg -i build_deb/noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f
```

安装脚本会自动将程序部署至 `/opt/noxfort-monitor/`、建立软链接 `/usr/local/bin/noxfort-monitor`、配置桌面图标并激活 Mosquitto MQTT 服务。

---

## 2. 独立 Systemd 服务（无头模式）

适用于云主机或工控无显示屏服务器：

### 2.1 服务单元文件 (`/etc/systemd/system/noxfort-monitor.service`)
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

### 2.2 启动与自启设置
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now noxfort-monitor
sudo journalctl -u noxfort-monitor -f
```

---

## 3. NGINX 反向代理与 SSL 加密

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
