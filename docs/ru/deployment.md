# 🚀 Руководство по развертыванию в продакшене

Документ описывает развертывание **Noxfort Monitor™ v2.0** с конфигурацией `.env` (после удаления `configs/config.yaml`), пакетом `.deb`, systemd headless, Docker Compose, обратным прокси, плановыми бэкапами и Prometheus.

⬅️ [Главный Хаб](../README.md) | 🏛️ [Архитектура](architecture.md) | 🖥️ [Десктоп](desktop_app.md)

---

## 1. Конфигурация окружения (`.env`)

Файл `configs/config.yaml` удален. Параметры запуска загружаются из `.env` функцией `LoadDotEnv()` (см. `.env.example`), а динамические настройки (URL брокера MQTT, оповещения, туннели) хранятся в SQL-таблице `settings`.

| Переменная | По умолчанию | Описание |
|---|---|---|
| `MONITOR_MQTT_STRICT` | `true` | Отбрасывает MQTT-сообщения (`noxfort/telemetry/#`) от неизвестных или `PAUSED` узлов. |
| `TRUSTED_PROXIES` | *(пусто)* | Список CIDR/IP доверенных прокси для `X-Real-IP`, `X-Forwarded-For` и `X-Forwarded-Proto`. |
| `MONITOR_USE_HTTPS` | `false` | Включает `Secure` cookies и определение HTTPS за TLS-прокси. |
| `MONITOR_RETENTION_DAYS` | `90` | Срок хранения записей до очистки `RetentionManager` (раз в 24ч). |
| `MONITOR_BACKUP_INTERVAL_HOURS` | `24` | Интервал автоматических горячих бэкапов `BackupManager` (хранится до `7` копий). |
| `MONITOR_DB_DRIVER` / `MONITOR_DB_*` | `sqlite` | Параметры прямого подключения к PostgreSQL при старте. |

---

## 2. Пакет Debian (`.deb`) и Docker Compose

```bash
# Установка .deb
make deb
make install-deb        # Или: sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb && sudo apt-get install -f
make uninstall-deb

# Docker Compose (--headless + Mosquitto + опциональный PostgreSQL)
cp .env.example .env
docker compose up -d --build
./scripts/setup_mqtt_auth.sh
```

---

## 3. Выделенный сервис Systemd (Headless)

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

## 4. Обратный прокси (Caddy / NGINX), бэкапы и `/metrics`

* **Вариант A — Caddy (`caddy/Caddyfile`)**: `make caddy-build && make services-start` ([Интеграция Caddy](caddy_integration.md)).
* **Вариант B — NGINX**: Добавьте IP NGINX в `TRUSTED_PROXIES` и проксируйте `/api/pair`, `/api/telemetry` или `/`:

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

* **Бэкапы, ретеншн и Prometheus**: `BackupManager` создает снимки `VACUUM INTO` / `pg_dump -F c` (с поддержкой `/api/settings/database/backups/download` и `/restore`), `RetentionManager` очищает старые логи, а `GET /metrics` отдает метрики для Prometheus.
