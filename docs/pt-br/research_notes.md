# 🔬 Notas de Pesquisa e Decisões Técnicas: Noxfort Monitor™

Este documento consolida o registro de decisões arquiteturais (ADRs), benchmarks e o roadmap de pesquisa técnica do **Noxfort Monitor™**.

⬅️ [Central de Documentação](README.md) | 🏛️ [Arquitetura](architecture.md) | 🗄️ [Banco de Dados](database.md) | 🌐 [Acesso Remoto](remote_access.md)

---

## 1. Persistência: Da Dependência de CGO à Arquitetura Dual-Engine

### Contexto
As versões iniciais dependiam exclusivamente do driver `mattn/go-sqlite3`, exigindo um compilador C (CGO) para compilar o executável. Isso gerava atrito severo em pipelines de CI/CD, compilação cruzada e empacotamento.

### Decisão
1. **Migração para Pure-Go SQLite**: Adoção do `modernc.org/sqlite`, viabilizando compilação 100% nativa em Go puro.
2. **Evolução para Dual-Engine (SQLite + PostgreSQL)**: Em plantas industriais de grande porte, múltiplos operadores acessam a interface simultaneamente, causando contenção de locks (`database is locked`) no SQLite em escritas de alta frequência.
3. **Hot-Reload com Zero Downtime**: Criação do [`DBManager`](database.md) para permitir a alternância dinâmica entre SQLite e PostgreSQL sem derrubar a ingestão de telemetria.

---

## 2. Interface Desktop: Da Systray Simples ao Wails v2

### Contexto
A primeira versão utilizava um ícone de bandeja simples (`getlantern/systray`) que disparava o navegador padrão do sistema operacional, causando conflitos de sessão e falta de atalhos dedicados.

### Decisão
Migração para o **Wails v2** ([`desktop_app.md`](desktop_app.md)):
* **WebKitGTK Nativo**: Renderização veloz e integrada de HTML5/Bootstrap sem o consumo excessivo de memória do Chromium/Electron.
* **Ciclo de Vida Integrado**: A bandeja do sistema foi reescrita dentro do loop de eventos do Wails, permitindo minimizar para a bandeja ao fechar a janela (`HideWindowOnClose`).
* **Isolamento de Sessão**: Interceptador de cabeçalhos sincroniza o cookie `noxfort_session` na memória do runtime desktop.
* **Compatibilidade Headless**: Adição da flag `--headless` permitindo rodar o mesmo binário como daemon em servidores Linux sem servidor gráfico X11/Wayland.

---

## 3. Conectividade WAN: DuckDNS + Caddy vs Túnel Ngrok

### Contexto
Agentes de telemetria externos (**Carina**, **Synapse**) e nós IoT operam fora da rede local do servidor. Inicialmente, o Monitor adotava apenas o túnel reverso do **Ngrok**. Porém, planos gratuitos impõem limites de largura de banda, expiração de túneis e dependência de infraestrutura proprietária de terceiros.

### Decisão
Evolução para arquitetura de conectividade dupla no subsistema [`internal/tunnel`](remote_access.md) e [`caddy/`](caddy_integration.md):
1. **DuckDNS + Caddy Gateway (Primário / Recomendado)**:
   * Atualizador dinâmico nativo em Go sincroniza IPv4 e IPv6 com o DuckDNS.
   * Contêiner Caddy automatiza a emissão de certificados SSL/TLS via **desafio ACME DNS-01**.
   * Não requer abertura de porta 80, terminando HTTPS com segurança na porta padrão 443 e expondo WebSockets seguros (`wss:///mqtt`) para o Mosquitto.
2. **Túnel Reverso Ngrok (Contingência / Zero-Forwarding)**:
   * Mantido como alternativa para redes com Carrier-Grade NAT (CGNAT) severo onde não é possível abrir portas no roteador de borda.

---

## 4. Roadmap e Linhas Futuras de Pesquisa

### 4.1 Federação de Servidores com gRPC
Para clusters distribuídos e comunicação matriz-filial de múltiplos servidores Monitor, está planejada a adição de replicação via **gRPC / Protocol Buffers**:
* Redução drástica da sobrecarga de payload em links satelitais e de baixa latência.
* Contratos imutáveis e fortemente tipados para sincronização da trilha de auditoria.

### 4.2 Detecção Preditiva de Anomalias
Atualmente, alertas utilizam regras estáticas de limiar (ex: temperatura crítica). Está em estudo a incorporação de modelos leves de séries temporais em Go (como médias móveis ponderadas e inferência embarcada ONNX) para prever falhas antes que os limites críticos de operação sejam atingidos.
