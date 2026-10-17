<div align="center">

<img src="../assets/monitor-logo.png" alt="Логотип Noxfort Monitor" width="120" />

# Noxfort Monitor™ — Комплект технической документации
### Архитектура системы, сбор телеметрии и двухдвижковая персистентность
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Active-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Платформа](https://img.shields.io/badge/Platform-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![Лицензия: AGPL-3.0-or-later](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](../../LICENSE)

---

🌐 **Языки / Переводы:** **[🇺🇸 English](../en/README.md)** • **[🇧🇷 Português](../pt-br/README.md)** • **[🇪🇸 Español](../es/README.md)** • **[🇫🇷 Français](../fr/README.md)** • **[🇷🇺 Русский](README.md)** • **[🇨🇳 简体中文](../zh/README.md)** • **[📚 Главный Хаб](../README.md)**

---

</div>

## Добро пожаловать в официальную документацию

В данном каталоге представлен полный комплект технической документации на **русском языке** для **Noxfort Monitor™ v2.0** — промышленной событийно-ориентированной платформы сбора телеметрии, мониторинга и реагирования на инциденты, разработанной на Go для распределенных систем, автономных агентов (**Carina**, **Synapse**) и промышленных IoT-узлов.

## Содержание технических руководств

| Документ | Раздел & Назначение | Основные темы |
|---|---|---|
| 📖 **[Архитектура системы](architecture.md)** | Базовая архитектура | Событийно-ориентированная архитектура (EDA), параллелизм на горутинах, слои SOLID и внедрение зависимостей в корне композиции. |
| 📡 **[API и протоколы телеметрии](api_reference.md)** | Протоколы обмена | Асинхронный прием MQTT (`:1883`), REST `POST /api/telemetry` (`:22100`), единая JSON-схема `IncomingEvent`, эндпоинты контактов, регистрация и наблюдаемость. |
| 🗄️ **[База данных и Dual-Engine](database.md)** | Хранение данных | Горячее переключение в рантайме через `DBManager`, PostgreSQL и чистый Go-драйвер SQLite без CGO, миграция без простоя и горячее резервное копирование. |
| 🔐 **[Безопасность и RBAC](security.md)** | Контроль доступа | Ролевой доступ (`RoleAdmin`, `RoleOperator`), промышленная сложность паролей, индикатор надежности, хеширование с солью и безопасный bootstrap. |
| 🌐 **[Удаленный доступ и WAN](remote_access.md)** | Связь через WAN | Динамический DNS DuckDNS, автоматический TLS через Caddy, резервный туннель Ngrok через промышленные файрволы и CGNAT. |
| 🛡️ **[Интеграция Caddy и DuckDNS](caddy_integration.md)** | Шлюз периметра сети | Контейнеризированный обратный прокси Caddy, автоматическая валидация ACME DNS-01, изоляция веб-интерфейса и шифрование WSS. |
| 🖥️ **[Десктоп и Headless](desktop_app.md)** | Интерфейс и запуск | Wails v2 + WebKitGTK, блокировка единого экземпляра через сокет Unix IPC, системный трей и режим сервера `--headless`. |
| 🔍 **[Аудиторский след и SLA](audit_trail.md)** | Прослеживаемость | Модель тройного аудита: журналы доступа (`SecurityAuditLog`), проверка SLA доставки оповещений (`AlertDispatchLog`) и время простоя. |
| 🚀 **[Руководство по развертыванию](deployment.md)** | Эксплуатация и DevOps | Сервис systemd в headless-режиме, обратный прокси (Caddy или NGINX) и пакетная установка Debian `.deb`. |
| 🧪 **[Тестирование и контроль качества](testing.md)** | Проверка и QA | Набор модульных тестов с моками на Go, ручная проверка E2E через `mosquitto_pub` / `curl` и системная наблюдаемость (`/healthz`, `/metrics`). |
| 🛠️ **[Руководство для разработчиков](developer_guides.md)** | Сборка и расширение | Локальная среда разработки, сборка Wails v2, создание миграций БД, добавление новых каналов оповещений и стандарты кода. |
| 🔬 **[Заметки об исследованиях (ADR)](research_notes.md)** | Архитектурные решения | Обоснование выбора чистого Go SQLite (modernc.org), гибридного протокола DuckDNS + Caddy ACME, Wails v2 и модели отказоустойчивости. |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Промышленная телеметрия и наблюдаемость • Noxfort Monitor™ Server v2.0</i><br/>
  <small>Лицензировано по <a href="../../LICENSE">GNU Affero General Public License v3.0 или более поздней версии</a> (SPDX: <code>AGPL-3.0-or-later</code>). © 2026 Noxfort Systems.</small>
</div>
