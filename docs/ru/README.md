<div align="center">

<img src="../assets/monitor-logo.png" alt="Логотип Noxfort Monitor" width="120" />

# Noxfort Monitor™ — Комплект технической документации
### Архитектура системы, сбор телеметрии и двухдвижковая персистентность
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Active-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Платформа](https://img.shields.io/badge/Platform-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![Лицензия](https://img.shields.io/badge/License-AGPL_v3-blue?style=flat)](../../LICENSE)

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
| 📡 **[API и протоколы телеметрии](api_reference.md)** | Протоколы обмена | Асинхронный прием MQTT (`:1883`), REST `POST /api/telemetry` (`:22100`), единая JSON-схема `IncomingEvent` и административные маршруты. |
| 🗄️ **[База данных и Dual-Engine](database.md)** | Хранение данных | Горячее переключение в рантайме через `DBManager`, PostgreSQL и чистый Go-драйвер SQLite без CGO, миграция без простоя. |
| 🔐 **[Безопасность и RBAC](security.md)** | Контроль доступа | Ролевой доступ (`RoleAdmin`, `RoleOperator`), хеширование паролей с солью, cookies сессий и идемпотентная инициализация. |
| 🌐 **[Удаленный доступ и Ngrok](remote_access.md)** | Связь через WAN | Обратный туннель TLS через промышленные файрволы и CGNAT, поддержка статических доменов и сбор телеметрии с внешних узлов. |
| 🖥️ **[Десктоп и Headless](desktop_app.md)** | Интерфейс и запуск | Wails v2 + WebKitGTK, блокировка единого экземпляра через сокет Unix IPC, системный трей и режим сервера `--headless`. |
| 🔍 **[Аудиторский след и SLA](audit_trail.md)** | Прослеживаемость | Модель тройного аудита: журналы доступа (`SecurityAuditLog`), проверка SLA доставки оповещений (`AlertDispatchLog`) и время простоя. |
| 🚀 **[Руководство по развертыванию](deployment.md)** | Эксплуатация и DevOps | Сервис systemd в headless-режиме, обратный прокси NGINX с SSL и пакетная установка Debian `.deb`. |
| 🧪 **[Тестирование и контроль качества](testing.md)** | Проверка и QA | Набор модульных тестов с моками на Go, ручная проверка E2E через `mosquitto_pub` / `curl` и диагностика каналов. |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Промышленная телеметрия и наблюдаемость • Noxfort Monitor™ Server v2.0</i>
</div>
