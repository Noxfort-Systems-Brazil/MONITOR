# 🚀 生产环境部署指南

本文档提供在工业级与企业级生产环境中部署 **Noxfort Monitor™ v2.0** 的详细步骤，包括 Debian 软件包 (`.deb`) 安装方式、基于 Systemd 的无界面（Headless）服务运行，以及反向代理网关配置（Caddy 或 NGINX）。

⬅️ [文档中心](../README.md) | 🏛️ [系统架构](architecture.md) | 🖥️ [桌面应用](desktop_app.md)

---

## 1. 通过 Debian 软件包安装 (`.deb`)

```bash
# 构建 .deb 安装包（产物将生成在仓库根目录下：noxfort-monitor_2.0.1_amd64.deb）
make deb

# 方式 A：通过 Makefile 自动安装或升级
make install-deb

# 方式 B：在目标 Ubuntu/Debian 机器上通过 dpkg 直接安装
sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f

# 卸载程序：
make uninstall-deb
```

安装脚本会自动将程序部署至 `/opt/noxfort-monitor/`、建立系统软链接 `/usr/local/bin/noxfort-monitor`、配置桌面图标并激活 Mosquitto MQTT 服务。

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

## 3. 反向代理与 HTTPS 网关 (Caddy 或 NGINX)

### 3.1 方案 A：Caddy Server 配合 DuckDNS（推荐方案）
系统内置容器化 Caddy 边缘网关，支持基于 **DNS-01 ACME** 的全自动证书签发与续期：
```bash
make caddy-build
make services-start
```
详细配置步骤请参阅 [Caddy 与 DuckDNS 集成指南](caddy_integration.md)。

### 3.2 方案 B：使用 NGINX 代理遥测接入
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

    # 代理外部边缘节点的遥测数据流
    location /api/telemetry {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 按安全设计，外部浏览器访问根路径直接返回 403 Forbidden
    location / {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
> [!NOTE]
> 外部 HTTP 端口已禁用直接通过网页浏览器访问后台仪表盘（返回 403 Forbidden）。图形界面管理操作仅允许在原生桌面客户端中运行。
