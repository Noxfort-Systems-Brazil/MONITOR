# Noxfort Monitor — Central de Documentação (Português BR)

Bem-vindo à suíte de documentação em Português (Brasil) do **Noxfort Monitor**.

## Guias Disponíveis

| Guia | Descrição |
| :--- | :--- |
| [Arquitetura do Sistema](./architecture.md) | Arquitetura modular em Go, modos (`desktop`, `--headless`, `--demo`), pipelines MQTT/HTTP e encerramento gracioso. |
| [Referência de API e Protocolos](./api_reference.md) | Tópicos `noxfort/telemetry/#`, pareamento Zero-Trust `/api/pair`, `/api/telemetry`, endpoints RBAC e `/metrics`. |
| [Arquitetura de Segurança](./security.md) | Perfil `OPERATOR` somente leitura, `PBKDF2`, `DBSessionStore`, bloqueio anti-força bruta, `TRUSTED_PROXIES` e redação de segredos. |
| [Banco de Dados e Ciclo de Vida](./database.md) | SQLite WAL + PostgreSQL em tempo real, `7` `ReloadableRepository`, `MigrateData` transacional, backup/restore e `RetentionManager`. |
| [Trilha de Auditoria](./audit_trail.md) | `SecurityAuditLog`, `AlertDispatchLog` (`telemetry_id`), `DeviceStateTransition`, acesso `Operator+` e retenção de 90 dias. |
| [Implantação e Operação](./deployment.md) | Variáveis `.env` (`configs/config.yaml` removido), Docker Compose, NGINX, backups automáticos e Prometheus. |
| [Proxy Reverso Caddy](./caddy_integration.md) | `caddy/Caddyfile`, `TRUSTED_PROXIES`, `MONITOR_USE_HTTPS` e testes HTTPS Zero-Trust. |
| [Acesso Remoto e Túneis](./remote_access.md) | DuckDNS e Ngrok em `/remote`, mascaramento de credenciais e onboarding remoto seguro. |
| [Aplicação Desktop e Modos](./desktop_app.md) | Wails GUI vs. `--demo` vs. `--headless`, `forbidden.html`, polling `/api/dashboard/summary` e troca de senha. |
| [Testes e Verificação](./testing.md) | `make test` (Go + Vitest), `go vet`, `mosquitto_pub`, testes `curl` Zero-Trust e `make record-demo`. |
| [Guias de Desenvolvimento](./developer_guides.md) | Comandos do `Makefile`, `ReloadableRepository`, helpers HTTP de segurança, `internal/metrics` e desligamento limpo. |
| [Decisões Arquiteturais (ADRs)](./research_notes.md) | Histórico técnico sobre `.env`, pareamento Zero-Trust, sessões hasheadas no banco, WAL/retenção e rastreabilidade. |
