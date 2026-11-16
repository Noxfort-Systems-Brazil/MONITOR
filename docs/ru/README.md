<div align="center">

<img src="../assets/monitor-logo.png" alt="Логотип Noxfort Monitor" width="120" />

# Noxfort Monitor™ — Комплект технической документации
### Архитектура системы, Zero-Trust телеметрия и двухдвижковая персистентность
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

В данном каталоге представлен комплект технической документации на **русском языке** для **Noxfort Monitor™ v2.0** — промышленной событийно-ориентированной платформы сбора Zero-Trust телеметрии, мониторинга и реагирования на инциденты на Go для распределенных систем, автономных агентов (**Carina**, **Synapse**) и промышленных IoT-узлов.

## Содержание технических руководств

| Документ | Раздел & Назначение | Основные темы |
|---|---|---|
| 📖 **[Архитектура системы](architecture.md)** | Базовая архитектура | EDA, горутины, `BufferedTelemetryWriter`, режимы Desktop / `--headless` / `--demo` и корректное завершение (`AlertService.Shutdown`). |
| 📡 **[API и протоколы телеметрии](api_reference.md)** | Протоколы обмена | Асинхронный MQTT `noxfort/telemetry/#` (`MONITOR_MQTT_STRICT`), Zero-Trust `POST /api/pair` и `POST /api/telemetry` (`:22100`), `IncomingEvent`, RBAC-роуты и `/metrics`. |
| 🗄️ **[База данных и Dual-Engine](database.md)** | Хранение данных | SQLite WAL + PostgreSQL с горячим переключением (`DBManager`), `7` `ReloadableRepository`, транзакционный `MigrateData` (9 таблиц), бэкапы/восстановление и `RetentionManager`. |
| 🔐 **[Безопасность и RBAC](security.md)** | Контроль доступа | Строго читающий `OPERATOR` (`403` / `forbidden.html`), `PBKDF2Hasher` (100 тыс. итераций), `LoginAttemptTracker` (`429`), `DBSessionStore` (`SHA-256`), `TRUSTED_PROXIES` и маскировка в HTML. |
| 🌐 **[Удаленный доступ и WAN](remote_access.md)** | Связь через WAN | DuckDNS и Ngrok на `/remote`, сокрытие секретов в HTML и удаленная Zero-Trust интеграция узлов. |
| 🛡️ **[Интеграция Caddy и DuckDNS](caddy_integration.md)** | Шлюз периметра сети | Автоматический TLS ACME DNS-01, заголовки `X-Forwarded-*` с `TRUSTED_PROXIES`, порт 443 и шифрованный WSS (`/mqtt`). |
| 🖥️ **[Десктоп, Headless и Demo](desktop_app.md)** | Интерфейс и запуск | Wails v2 + WebKitGTK, блокировка экземпляра IPC Unix, режимы `--headless` и `--demo`, polling `/api/dashboard/summary` и страница `forbidden.html`. |
| 🔍 **[Аудиторский след и SLA](audit_trail.md)** | Прослеживаемость | Тройной аудит (`SecurityAuditLog`, `AlertDispatchLog` со связью `telemetry_id`, `DeviceStateTransition`), доступ `Operator+` и автоочистка за 90 дней. |
| 🚀 **[Руководство по развертыванию](deployment.md)** | Эксплуатация и DevOps | Переменные `.env` (`configs/config.yaml` удален), systemd headless, Docker Compose, Caddy/NGINX, `.deb`-пакет, бэкапы и `/metrics`. |
| 🧪 **[Тестирование и контроль качества](testing.md)** | Проверка и QA | Go + Vitest (`make test`), моки, E2E-проверка `mosquitto_pub` (`noxfort/telemetry/#`) / `curl` (`/api/pair` + `X-Device-Token`) и `make record-demo`. |
| 🛠️ **[Руководство для разработчиков](developer_guides.md)** | Сборка и расширение | Локальная среда, контракт `ReloadableRepository` (`SetDB(db, driver)`), `internal/metrics`, HTTP-утилиты безопасности и жизненный цикл `bgCtx`. |
| 🔬 **[Заметки об исследованиях (ADR)](research_notes.md)** | Архитектурные решения | Отказ от CGO и `config.yaml`, Zero-Trust сопряжение, хешированные сессии в БД, WAL/ретеншн и нативные метрики Prometheus. |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>Промышленная телеметрия и наблюдаемость • Noxfort Monitor™ Server v2.0</i><br/>
  <small>Лицензировано по <a href="../../LICENSE">GNU Affero General Public License v3.0 или более поздней версии</a> (SPDX: <code>AGPL-3.0-or-later</code>). © 2026 Noxfort Systems.</small>
</div>
