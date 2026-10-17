<div align="center">

<img src="../assets/monitor-logo.png" alt="Noxfort Monitor 标志" width="120" />

# Noxfort Monitor™ — 官方技术文档套件
### 核心系统架构、遥测数据摄取与双引擎持久化机制
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

本目录包含 **Noxfort Monitor™ v2.0** 的完整**简体中文**技术文档库。Noxfort Monitor 是由 Noxfort Systems 研发的高性能工业级事件驱动可观测性平台，专为分布式系统、自主智能体（如 **Carina** 与 **Synapse**）以及各类工业物联网边缘节点而设计。

## 专业技术指南目录

| 文档 | 领域与范围 | 核心内容 |
|---|---|---|
| 📖 **[系统架构核心设计](architecture.md)** | 系统设计蓝图 | 事件驱动架构 (EDA)、Goroutine 高并发模型、SOLID 分层设计及组合根依赖注入机制。 |
| 📡 **[API 与遥测接入协议](api_reference.md)** | 工业通信协议 | 非阻塞 MQTT 协议摄取 (`:1883`)、REST `POST /api/telemetry` (`:22100`)、应急响应团队接口、用户注册及系统可观测性。 |
| 🗄️ **[双数据库引擎与持久化](database.md)** | 关系数据持久化 | 运行时热切换管理器 (`DBManager`)、PostgreSQL 与免 CGO 的纯 Go SQLite、异构无损数据迁移与自动化热备份。 |
| 🔐 **[系统安全与 RBAC 控制](security.md)** | 安全与权限体系 | 基于角色的访问控制 (`RoleAdmin` 与 `RoleOperator`)、工业级密码复杂度校验、加盐密码散列、会话凭证及安全初始化。 |
| 🌐 **[远程访问与广域网接入](remote_access.md)** | 广域网边缘穿透 | DuckDNS 动态域名同步、Caddy 自动 HTTPS 终结及 Ngrok 备用反向出站隧道。 |
| 🛡️ **[Caddy 反向代理与 DuckDNS](caddy_integration.md)** | 边缘接入网关 | 容器化 Caddy 网关、基于 DNS-01 ACME 的全自动证书签发、仪表盘外网隔离及加密 WebSockets (WSS)。 |
| 🖥️ **[桌面应用与无头守护模式](desktop_app.md)** | 桌面运行与服务 | 基于 Wails v2 与原生 WebKitGTK、Unix IPC 套接字单实例锁、系统托盘及 `--headless` 无头守护模式。 |
| 🔍 **[审计日志与 SLA 追踪](audit_trail.md)** | 合规可追溯性 | 三重不可变审计流：安全访问审计 (`SecurityAuditLog`)、报警推送 SLA 履约日志 (`AlertDispatchLog`) 及设备停机追踪。 |
| 🚀 **[生产环境部署指南](deployment.md)** | 运维实施与部署 | 独立 Systemd 无头服务单元、反向代理配置（Caddy 或 NGINX）及 Debian 原生 `.deb` 安装包。 |
| 🧪 **[测试规范与质量验证](testing.md)** | 质量保证与测试 | 包含 Mock 内存仓储的自动化单元测试套件、端到端手动验证及可观测性接口诊断 (`/healthz`, `/metrics`)。 |
| 🛠️ **[高级开发者指南](developer_guides.md)** | 构建与功能扩展 | 本地开发环境配置、Wails v2 编译构建、热重载仓储模式、添加新告警通道及代码规范。 |
| 🔬 **[技术预研与架构决策 (ADR)](research_notes.md)** | 技术选型与演进 | 纯 Go SQLite 选型、DuckDNS + Caddy 广域网方案、Wails v2 桌面设计及未来 gRPC 集群演进路线。 |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>工业遥测与分布式可观测性 • Noxfort Monitor™ Server v2.0</i><br/>
  <small>基于 <a href="../../LICENSE">GNU Affero General Public License v3.0 或更高版本</a> (SPDX: <code>AGPL-3.0-or-later</code>) 授权。© 2026 Noxfort Systems。</small>
</div>
