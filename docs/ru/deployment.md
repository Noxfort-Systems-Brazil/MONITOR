# 🚀 Руководство по развертыванию в продакшене

Данный документ содержит пошаговые инструкции по развертыванию **Noxfort Monitor™ v2.0** в промышленной эксплуатации, включая установку systemd-сервиса в headless-режиме, настройку обратного прокси (Caddy или NGINX) и установку пакета Debian (`.deb`).

⬅️ [Главный Хаб](../README.md) | 🏛️ [Архитектура](architecture.md) | 🖥️ [Десктоп](desktop_app.md)

---

## 1. Установка пакета Debian (`.deb`)

```bash
# Сборка установочного пакета .deb (создается в корне репозитория как noxfort-monitor_2.0.1_amd64.deb)
make deb

# Вариант A: Автоматическая установка/обновление через Makefile
make install-deb

# Вариант B: Прямая установка через dpkg на целевом сервере Ubuntu/Debian
sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f

# Для удаления:
make uninstall-deb
```

Пакет устанавливает бинарные файлы в каталог `/opt/noxfort-monitor/`, создает симлинк `/usr/local/bin/noxfort-monitor`, регистрирует системные ярлыки и настраивает брокер Mosquitto MQTT.

---

## 2. Выделенный сервис Systemd (Headless)

Рекомендуется для облачных виртуальных машин и серверов без монитора:

### 2.1 Файл юнита (`/etc/systemd/system/noxfort-monitor.service`)
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

### 2.2 Запуск и включение автозагрузки
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now noxfort-monitor
sudo journalctl -u noxfort-monitor -f
```

---

## 3. Обратный прокси и HTTPS-шлюз (Caddy или NGINX)

### 3.1 Вариант A: Сервер Caddy с DuckDNS (Рекомендуется)
Экосистема включает контейнеризированный пограничный шлюз Caddy с автоматическим получением сертификатов **DNS-01 ACME**:
```bash
make caddy-build
make services-start
```
См. подробное руководство в [Интеграция Caddy и DuckDNS](caddy_integration.md).

### 3.2 Вариант B: NGINX для приема телеметрии
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

    # Проксирование телеметрии с внешних узлов Edge
    location /api/telemetry {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Запросы веб-браузера к корню возвращают 403 Forbidden по соображениям безопасности
    location / {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
> [!NOTE]
> Доступ через обычный браузер отключен на внешних HTTP-портах (возвращается 403 Forbidden). Управление интерфейсом осуществляется исключительно через нативное десктопное приложение.
