# 🚀 Руководство по развертыванию в продакшене

Данный документ содержит пошаговые инструкции по развертыванию **Noxfort Monitor™ v2.0** в промышленной эксплуатации, включая установку systemd-сервиса в headless-режиме, настройку обратного прокси NGINX с SSL и установку пакета Debian (`.deb`).

⬅️ [Главный Хаб](../README.md) | 🏛️ [Архитектура](architecture.md) | 🖥️ [Десктоп](desktop_app.md)

---

## 1. Установка пакета Debian (`.deb`)

```bash
# Сборка установочного пакета .deb
make deb

# Установка на целевой сервер Ubuntu/Debian
sudo dpkg -i build_deb/noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f
```

Пакет устанавливает бинарные файлы в каталог `/opt/noxfort-monitor/`, создает симлинк `/usr/local/bin/noxfort-monitor`, добавляет ярлыки и запускает брокер Mosquitto MQTT.

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

## 3. Обратный прокси NGINX с SSL-терминацией

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
