# 🔬 技术预研与架构决策记录 (ADR)：Noxfort Monitor™

本文档记录了 **Noxfort Monitor™** 的技术选型演进、基准考量、架构决策（ADR）以及未来技术演进路线图。

⬅️ [文档中心](README.md) | 🏛️ [系统架构](architecture.md) | 🗄️ [双引擎数据库](database.md) | 🌐 [远程访问](remote_access.md)

---

## 1. 持久层选型：从 CGO 依赖走向双引擎架构

### 背景
早期版本的 Monitor 深度绑定 `mattn/go-sqlite3`，编译时强依赖 C 编译器 (CGO)。这给跨平台编译、CI/CD 自动化流水线和轻量容器化带来了极大的构建负担。

### 决策
1. **拥抱纯 Go SQLite**：全面转向 `modernc.org/sqlite`，在嵌入式或边缘单机部署场景下实现 100% 纯 Go 静态编译。
2. **演进至双引擎架构（SQLite + PostgreSQL）**：在大型工控场景下，多个工程师可能并发访问监控后台，单文件 SQLite 高频并发写操作极易引发锁冲突 (`database is locked`)。
3. **零停机热重载**：研发了 [`DBManager`](database.md)，支持管理员在 Web/桌面控制台中随时一键热切换 SQLite 与 PostgreSQL，遥测数据接收全程不中断。

---

## 2. 桌面客户端：从简单系统托盘到 Wails v2

### 背景
初始版本仅通过简易系统托盘 (`getlantern/systray`) 调用系统默认浏览器。实际应用中经常出现与其他 Web 标签页的会话冲突、缺乏专属快捷键、以及用户体验脱节的问题。

### 决策
全面重构为基于 **Wails v2** 的现代化工业级桌面客户端 ([`desktop_app.md`](desktop_app.md))：
* **原生 WebKitGTK 引擎**：高效渲染 HTML5/Bootstrap 界面，内存占用远低于传统 Chromium/Electron 方案。
* **统一的生命周期管理**：系统托盘无缝集成至 Wails 事件循环，支持点击关闭最小化到托盘 (`HideWindowOnClose`) 与单机一键呼出。
* **独立的会话隔离**：通过底层请求头拦截器在桌面内存中隔离并维护 `noxfort_session` Cookie。
* **兼顾无界面服务 (Headless)**：提供 `--headless` 启动参数，使得同一份编译产物即可作为无界面的 Linux 后台服务稳定运行。

---

## 3. 广域网连接：DuckDNS + Caddy 边缘网关 与 Ngrok 反向隧道

### 背景
工业边缘代理（**Synapse**、**Carina**）与外部 IoT 设备通常位于服务器局域网之外。最初 Monitor 仅依赖 **Ngrok** 出站隧道来穿透 CGNAT 与防火墙。然而，第三方隧道免费服务存在带宽受限、会话中断和对外部云服务强依赖等问题。

### 决策
将 [`internal/tunnel`](remote_access.md) 与 [`caddy/`](caddy_integration.md) 子系统升级为双模式互补架构：
1. **DuckDNS 动态解析 + 容器化 Caddy 网关（首选 / 推荐方案）**：
   * 原生 Go 驱动定时将公网 IPv4 与 IPv6 动态同步至 DuckDNS。
   * 容器化 Caddy 作为前置边缘网关，调用 DuckDNS API 自动执行 **ACME DNS-01 验证**。
   * 完全免去为了申请 TLS 证书而对外开放 80 端口的要求，对外统一提供 443 端口加密接入。
   * 负责反向代理遥测接口 `POST /api/telemetry` 至 22100 端口，并为 Mosquitto 提供加密 WebSockets (`wss://domain.duckdns.org/mqtt`)。
2. **Ngrok 反向隧道（备选 / 零端口映射方案）**：
   * 保留作为二级免配置驱动，专用于电信级严格 CGNAT 或无法进行任何端口映射的复杂网络环境。

---

## 4. 技术路线图与前瞻研究

### 4.1 gRPC 多节点集群联邦
虽然 MQTT 与 HTTP REST 完美胜任设备遥测，但在多服务器集群联邦（如主备热备、总部与分厂层级汇聚）中，将引入 **gRPC / Protocol Buffers**：
* 显著降低在受限专网或卫星通信链路中的网络开销。
* 为多节点审计日志同步提供强类型、不可变契约。

### 4.2 预测性异常分析
目前的告警系统基于静态阈值与规则（例如 `temperature > 80°C`）。我们正在评估基于 Go 的轻量级时间序列预测算法（如指数移动加权平均和嵌入式 ONNX 运行时），在达到故障临界点前主动识别潜在异常隐患。
