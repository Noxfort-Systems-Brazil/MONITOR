<div align="center">

<img src="../assets/monitor-logo.png" alt="Noxfort Monitor 标志" width="120" />

# Noxfort Monitor™ — 官方技术文档套件
### 核心系统架构、Zero-Trust 遥测摄取与双引擎持久化机制
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Active-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![License: AGPL-3.0-or-later](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](../../LICENSE)

---

🌐 **语言 / Translations:** **[🇺🇸 English](../en/README.md)** • **[🇧🇷 Português](../pt-br/README.md)** • **[🇪🇸 Español](../es/README.md)** • **[🇫🇷 Français](../fr/README.md)** • **[🇷🇺 Русский](../ru/README.md)** • **[🇨🇳 简体中文](README.md)** • **[📚 文档中心](../README.md)**

---

</div>

## 欢迎访问官方技术文档中心

本目录包含 **Noxfort Monitor™ v2.0** 的**简体中文**技术文档库 — 基于 Go 语言开发的工业级事件驱动可观测性、实时 Zero-Trust 遥测摄取与告警响应平台，适用于分布式系统、自主智能体（**Carina**、**Synapse**）及工业物联网边缘节点。

## 专业技术指南目录

| 文档 | 领域与范围 | 核心内容 |
|---|---|---|
| 📖 **[系统架构核心设计](architecture.md)** | 系统设计蓝图 | 事件驱动架构 (EDA)、Goroutine、`BufferedTelemetryWriter`、Desktop / `--headless` / `--demo` 模式与优雅停机 (`AlertService.Shutdown`)。 |
| 📡 **[API 与遥测接入协议](api_reference.md)** | 工业通信协议 | 异步 MQTT `noxfort/telemetry/#` (`MONITOR_MQTT_STRICT`)、Zero-Trust `POST /api/pair` 与 `POST /api/telemetry` (`:22100`)、`IncomingEvent`、RBAC 路由及 `/metrics`。 |
| 🗄️ **[双数据库引擎与持久化](database.md)** | 关系数据持久化 | SQLite WAL + PostgreSQL 热切换 (`DBManager`)、`7` 个 `ReloadableRepository`、单事务 9 表 `MigrateData`、热备份/验证恢复与 `RetentionManager`。 |
| 🔐 **[系统安全与 RBAC 控制](security.md)** | 安全与权限体系 | 严格只读 `OPERATOR`（`403` JSON / `forbidden.html`）、`PBKDF2Hasher`（10万次迭代）、`LoginAttemptTracker`（`429`）、`DBSessionStore`（`SHA-256`）、`TRUSTED_PROXIES` 与 HTML 脱敏。 |
| 🌐 **[远程访问与广域网接入](remote_access.md)** | 广域网边缘穿透 | DuckDNS 动态同步与 Ngrok 反向隧道、`/remote` 令牌脱敏与只读控制、Zero-Trust 远程接入最佳实践。 |
| 🛡️ **[Caddy 反向代理与 DuckDNS](caddy_integration.md)** | 边缘接入网关 | 自动 TLS（ACME DNS-01）、配合 `TRUSTED_PROXIES` 传递 `X-Forwarded-*` 请求头、443 端口及加密 WSS (`/mqtt`)。 |
| 🖥️ **[桌面应用、Headless 与 Demo 模式](desktop_app.md)** | 桌面运行与服务 | Wails v2 + WebKitGTK、Unix IPC 单实例锁、`--headless` 与 `--demo` 模式、`/api/dashboard/summary` 轮询与 `forbidden.html` 页面。 |
| 🔍 **[审计日志与 SLA 追踪](audit_trail.md)** | 合规可追溯性 | 三重审计流（`SecurityAuditLog`、关联 `telemetry_id` 的 `AlertDispatchLog`、`DeviceStateTransition`）、`Operator+` 只读查阅与 90 天自动清理。 |
| 🚀 **[生产环境部署指南](deployment.md)** | 运维实施与部署 | `.env` 配置（已移除 `configs/config.yaml`）、Systemd 无头服务、Docker Compose、Caddy/NGINX、`.deb` 安装包、定时备份与 `/metrics`。 |
| 🧪 **[测试规范与质量验证](testing.md)** | 质量保证与测试 | Go + Vitest (`make test`)、内存 Mock、MQTT (`noxfort/telemetry/#`) 与 Zero-Trust HTTP (`curl`) 端到端测试及 `make record-demo`。 |
| 🛠️ **[高级开发者指南](developer_guides.md)** | 构建与功能扩展 | `Makefile` 指令、`ReloadableRepository` (`SetDB(db, driver)`) 契约、`internal/metrics`、HTTP 安全辅助函数与 `bgCtx` 生命周期。 |
| 🔬 **[技术预研与架构决策 (ADR)](research_notes.md)** | 技术选型与演进 | 放弃 CGO 与 `config.yaml`、Zero-Trust 配对、数据库哈希会话、WAL/数据保留及原生 Prometheus 指标演进。 |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>工业遥测与分布式可观测性 • Noxfort Monitor™ Server v2.0</i><br/>
  <small>基于 <a href="../../LICENSE">GNU Affero General Public License v3.0 或更高版本</a> (SPDX: <code>AGPL-3.0-or-later</code>) 授权。© 2026 Noxfort Systems。</small>
</div>
