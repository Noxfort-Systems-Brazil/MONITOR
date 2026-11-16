# Noxfort Monitor — Trilha de Auditoria e Conformidade

O Noxfort Monitor mantém registros imutáveis de eventos de segurança, despachos de alertas e transições de conectividade em três tabelas especializadas, acessíveis para perfis `Operator+` em `/audit` e `/api/audit/*`.

---

## 1. Log de Auditoria de Segurança (`SecurityAuditLog`)

Registra ações administrativas, tentativas de autenticação e operações de infraestrutura em `security_audit_logs`.

* **Campos Reais (`internal/domain/audit.go`)**:
  * `id` (Chave primária)
  * `created_at` (Timestamp UTC)
  * `username` (Ator autenticado ou usuário alvo)
  * `action` (Identificador canônico da ação)
  * `details` (Detalhes contextuais e resultado)
  * `ip_address` (IP de origem resolvido por `ExtractClientIP` / `TRUSTED_PROXIES`)

* **Ações Auditadas**:
  * Autenticação e Bloqueio: `AUTH_LOGIN_SUCCESS`, `AUTH_LOGIN_FAILED`, `AUTH_LOGIN_LOCKED`
  * Usuários e RBAC: `USER_CREATE`, `USER_PASSWORD_CHANGED`, `USER_ROLE_UPDATED`, `USER_UNLOCKED`, `USER_DELETE`
  * Infraestrutura e Dispositivos: `DEVICE_*`, `CONTACT_*`, `SETTINGS_UPDATED`, `TUNNEL_*`, `DATABASE_*`, `POSTGRES_USER_PROVISIONED`

---

## 2. Log de Despacho de Alertas (`AlertDispatchLog`)

Rastreia cada tentativa de notificação executada pelo `AlertService` (Telegram, E-mail SMTP/TLS 465 e Webhooks) na tabela `alert_dispatch_logs`, contemplando retentativas assíncronas (`500ms`, `1500ms`) e drenagem no encerramento (`AlertService.Shutdown`).

* **Campos Reais**:
  * `id`
  * `telemetry_id` (Ligação direta ao registro `telemetry.id` obtido via `SaveEventReturningID`)
  * `channel` (`telegram`, `email`, `webhook`)
  * `recipient`
  * `role`
  * `status`
  * `error_reason`
  * `dispatched_at`

---

## 3. Transições de Estado de Dispositivos (`DeviceStateTransition`)

Registra mudanças de SLA e disponibilidade detectadas pelo `StateManager` e `Engine` (tick de `30s`, limiar offline de `5m`) na tabela `device_state_transitions`.

* **Campos Reais**:
  * `id`
  * `device_identifier`
  * `previous_state`
  * `new_state`
  * `duration_offline_sec`
  * `transition_at`

---

## 4. Acesso e Retenção de Logs

* **Acesso de Leitura**: Perfis `ADMIN` e `OPERATOR` (somente leitura) podem consultar `/audit`, `/api/audit/security`, `/api/audit/alerts` e `/api/audit/transitions`.
* **Expurgo Automático**: O `RetentionManager` remove a cada 24 horas registros mais antigos que `MONITOR_RETENTION_DAYS` (padrão **`90` dias**) de `security_audit_logs`, `alert_dispatch_logs`, `device_state_transitions` e `telemetry`.
