# 🚀 生产环境部署指南

本文档提供在工业与企业生产环境中部署 **Noxfort Monitor™ v2.0** 的最新指引：基于 `.env` 的配置体系（已移除 `configs/config.yaml`）、Debian 软件包 (`.deb`)、Systemd 无头服务、Docker Compose、反向代理与运维可观测性。

⬅️ [文档中心](../README.md) | 🏛️ [系统架构](architecture.md) | 🖥️ [桌面应用](desktop_app.md)

---

## 1. 环境变量配置 (`.env`)

旧版 `configs/config.yaml` 已彻底移除。核心引导参数通过 `LoadDotEnv()` 从 `.env` 读取（参见 `.env.example`），动态运行配置（MQTT Broker 地址、告警配置、远程隧道）则存储于 SQL `settings` 表。

| 变量名 | 默认值 | 说明 |
|---|---|---|
| `MONITOR_MQTT_STRICT` | `true` | 拒绝 `noxfort/telemetry/#` 上未注册或处于 `PAUSED` 状态设备的消息。 |
| `TRUSTED_PROXIES` | *(空)* | 逗号分隔的可信代理 CIDR/IP，用于验证 `X-Real-IP`、`X-Forwarded-For` 与 `X-Forwarded-Proto`。 |
| `MONITOR_USE_HTTPS` | `false` | 在 HTTPS 反向代理后启用 `Secure` Cookie 与 TLS 请求识别。 |
| `MONITOR_RETENTION_DAYS` | `90` | `RetentionManager` 清理 `telemetry`、审计及状态记录的保留天数（每 24h 执行）。 |
| `MONITOR_BACKUP_INTERVAL_HOURS` | `24` | `BackupManager` 自动热备份间隔小时数（最多保留 `7` 份归档）。 |
| `MONITOR_DB_DRIVER` / `MONITOR_DB_*` | `sqlite` | 可选 PostgreSQL 启动直连参数。 |

---

## 2. Debian 软件包 (`.deb`) 与 Docker Compose

```bash
# .deb 安装包
make deb
make install-deb        # 或: sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb && sudo apt-get install -f
make uninstall-deb

# Docker Compose（--headless + Mosquitto + 可选 PostgreSQL）
cp .env.example .env
docker compose up -d --build
./scripts/setup_mqtt_auth.sh
```

---

## 3. 独立 Systemd 服务（无头模式）

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

---

## 4. 反向代理 (Caddy / NGINX)、热备份与 `/metrics`

* **方案 A — Caddy (`caddy/Caddyfile`)**：执行 `make caddy-build && make services-start`（详见 [Caddy 集成指南](caddy_integration.md)）。
* **方案 B — NGINX**：将 NGINX IP 加入 `TRUSTED_PROXIES` 并透传 Zero-Trust 接口（`/api/pair`、`/api/telemetry` 或 `/`）：

```nginx
location / {
    proxy_pass http://127.0.0.1:22100;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

* **备份、保留与 Prometheus**：`BackupManager` 自动生成 `VACUUM INTO` / `pg_dump -F c` 快照并支持 `/api/settings/database/backups/download` 与 `/restore`，`RetentionManager` 定期裁剪过期数据，`GET /metrics` 提供 Prometheus 指标采集。
