<div align="center">

<img src="../assets/monitor-logo.png" alt="Noxfort Monitor 标志" width="120" />

# Noxfort Monitor™ — 官方技术文档套件
### 核心系统架构、遥测数据摄取与双引擎持久化机制
*Noxfort Systems — A State Of Art Company*

[![Status](https://img.shields.io/badge/Status-Active-brightgreen?style=flat&logo=github)](https://github.com/Noxfort-Systems-Brazil/MONITOR)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()
[![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=flat)]()
[![License](https://img.shields.io/badge/License-AGPL_v3-blue?style=flat)](../../LICENSE)

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
| 📡 **[API 与遥测接入协议](api_reference.md)** | 工业通信协议 | 非阻塞 MQTT 协议摄取 (`:1883`)、REST `POST /api/telemetry` (`:22100`)、统一 JSON 事件规范及管理接口。 |
| 🗄️ **[双数据库引擎与持久化](database.md)** | 关系数据持久化 | 运行时热切换管理器 (`DBManager`)、PostgreSQL 与免 CGO 的纯 Go SQLite、异构无损数据迁移与 SQL 方言适配。 |
| 🔐 **[系统安全与 RBAC 控制](security.md)** | 安全与权限体系 | 基于角色的访问控制 (`RoleAdmin` 与 `RoleOperator`)、加盐密码散列、会话凭证及幂等初始化机制。 |
| 🌐 **[远程访问与 Ngrok 穿透](remote_access.md)** | 广域网边缘穿透 | 基于 Ngrok 的安全反向 TLS 隧道，跨越工业防火墙与运营商 CGNAT，支持固定静态域名。 |
| 🖥️ **[桌面应用与无头守护模式](desktop_app.md)** | 桌面运行与服务 | 基于 Wails v2 与原生 WebKitGTK、Unix IPC 套接字单实例锁、系统托盘及 `--headless` 无头守护模式。 |
| 🔍 **[审计日志与 SLA 追踪](audit_trail.md)** | 合规可追溯性 | 三重不可变审计流：安全访问审计 (`SecurityAuditLog`)、报警推送 SLA 履约日志 (`AlertDispatchLog`) 及设备停机追踪。 |
| 🚀 **[生产环境部署指南](deployment.md)** | 运维实施与部署 | 独立 Systemd 无头服务单元、NGINX SSL 反向代理配置及 Debian 原生 `.deb` 安装包制作。 |
| 🧪 **[测试规范与质量验证](testing.md)** | 质量保证与测试 | 包含 Mock 内存仓储的自动化单元测试套件、基于 `mosquitto_pub` 与 `curl` 的端到端手动验证。 |

---

<div align="center">
  <img src="../assets/noxfort-logo.png" alt="Noxfort Systems Logo" width="45" /><br/>
  <b>Noxfort Systems</b> — <i>A State Of Art Company</i><br/>
  <i>工业遥测与分布式可观测性 • Noxfort Monitor™ Server v2.0</i>
</div>
