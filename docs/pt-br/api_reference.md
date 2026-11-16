# Noxfort Monitor — Referência de API e Protocolos

Este documento descreve os protocolos de ingestão de borda (MQTT e HTTP Zero-Trust), endpoints REST administrativos, permissões RBAC e métricas Prometheus expostas pelo Noxfort Monitor.

---

## 1. Protocolo de Telemetria MQTT

O cliente MQTT integrado em Go assina tópicos de telemetria de forma assíncrona em tempo real.

* **Endereço do Broker**: Configurável via `/settings` (ex.: `tcp://127.0.0.1:1883`), com reconexão automática a cada 5 segundos na inicialização e `Reconnect(brokerURL)` ao vivo ao salvar alterações.
* **Padrão de Tópico de Publicação**: `noxfort/telemetry/{identifier}`
* **Assinatura Wildcard**: `noxfort/telemetry/#`
* **Política Estrita de Dispositivos**: Controlada por `MONITOR_MQTT_STRICT` (padrão: `true`). Apenas dispositivos pré-cadastrados e `ACTIVE` são aceitos; mensagens de dispositivos não registrados ou `PAUSED` são descartadas.

### Formato de Payload (JSON — `IncomingEvent`)

```json
{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "CRITICAL",
  "message": "Superaquecimento detectado: Temperatura 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

* **`category`**: `HARDWARE` ou `SOFTWARE` (define o roteamento de alerta por perfil de contato).
* **`origin`**: Identificador único do dispositivo (ex.: `pump-01`, `carina`, `synapse`).
* **`level`**: `INFO`, `WARNING` ou `CRITICAL` (`INFO` com palavras de batimento atualiza `last_seen` sem gerar alertas; `CRITICAL` dispara persistência síncrona `SaveEventReturningID` e alertas ligados a `telemetry_id`).
* **`message`**: Descrição legível do evento.
* **`occurred_at`**: Carimbo UTC ISO-8601 (validado em `±300s` via HTTP).

---

## 2. Pareamento Zero-Trust e Ingestão HTTP

Dispositivos que utilizam HTTP/HTTPS passam por um fluxo Zero-Trust fora de banda antes de enviar telemetria.

### Pré-registro de Dispositivo (`POST /api/pair`)

* **Autenticação**: Pública (requisição inicial de pareamento).
* **Limites**: Payload máximo de 64 KB; fila limitada a `50` dispositivos `PENDING` (`429 Too Many Requests` se excedida).
* **Payload**:

```json
{
  "origin": "edge-gw-01",
  "name": "Edge Gateway 01"
}
```

* **Respostas**:
  * `202 Accepted`: Dispositivo colocado na fila em estado `PENDING`. **Nenhuma chave de API é retornada na resposta** (aprovação fora de banda: o administrador gera/copia em `/devices` a chave temporária `nox_live_` com TTL de 5 minutos).
  * `409 Conflict`: O dispositivo já está registrado como `ACTIVE`.

### Ingestão de Telemetria HTTP (`POST /api/telemetry`)

* **Autenticação**: Obrigatória via cabeçalho `X-Device-Token: <api_key>` ou `Authorization: Bearer <api_key>` (comparação em tempo constante).
* **Anti-Spoofing e Proteção contra Replay**:
  * `X-Device-Origin` é validado contra a origem/identificador registrado do dispositivo.
  * Carimbos `occurred_at` fora da janela de tolerância de `±300s` (`±5m`) são rejeitados.
* **Respostas**:
  * `202 Accepted`: Enfileirado via `BufferedTelemetryWriter` (com persistência síncrona `SaveEventReturningID` em transições de estado).
  * `401 Unauthorized`: Token ausente, expirado ou inválido.
  * `403 Forbidden`: Dispositivo em Modo Manutenção (`PAUSED`) ou divergência de origem.

---

## 3. Endpoints REST Administrativos e Web

Todos os endpoints protegidos utilizam sessões persistidas em cookie (`DBSessionStore`), rejeitam tokens via `?token=` na URL e aplicam RBAC (`ADMIN` vs. `OPERATOR` estritamente somente leitura).

| Endpoint | Método | Perfil | Descrição |
| :--- | :--- | :--- | :--- |
| `/api/auth/login` | `POST` | Público | Autentica usuário; aplica bloqueio `429` após 5 falhas em 15m (bloqueio de 5m). |
| `/api/auth/logout` | `POST` | Operator+ | Revoga o hash de sessão atual na tabela `sessions`. |
| `/api/dashboard/summary` | `GET` | Operator+ | Resumo JSON ao vivo (polling a cada 10s) nos 5 estados: `ONLINE`, `OFFLINE`, `PENDING`, `PAUSED`, `WAITING`. |
| `/devices/create` | `POST` | Admin | Cria dispositivo e exibe a chave de API uma única vez. |
| `/devices/update` | `POST` | Admin | Atualiza/renomeia dispositivo existente. |
| `/devices/toggle` | `POST` | Admin | Alterna Modo Manutenção (`PAUSED` / `ACTIVE`). |
| `/devices/rotate-key` | `POST` | Admin | Rotaciona a chave de API do dispositivo, invalidando a anterior. |
| `/devices/delete` | `POST` | Admin | Remove permanentemente um dispositivo. |
| `/api/users` | `GET` / `POST` | Admin | Lista usuários ou cria conta (`ADMIN` / `OPERATOR`). |
| `/api/users/password` | `POST` | Operator+ | Troca da própria senha (exige `current_password`). |
| `/api/users/role` | `POST` | Admin | Altera o perfil RBAC de um usuário. |
| `/api/users/unlock` | `POST` | Admin | Desbloqueia conta bloqueada pelo `LoginAttemptTracker`. |
| `/api/audit/security` | `GET` | Operator+ | Consulta registros de auditoria de segurança. |
| `/api/audit/alerts` | `GET` | Operator+ | Consulta logs de disparo de alertas (ligados via `telemetry_id`). |
| `/api/audit/transitions` | `GET` | Operator+ | Consulta transições históricas `ONLINE` / `OFFLINE`. |
| `/api/settings/database/backups` | `GET` | Admin | Lista backups quentes disponíveis (SQLite/PostgreSQL). |
| `/api/settings/database/backup` | `POST` | Admin | Gera backup imediato (`VACUUM INTO` / `pg_dump -F c`). |
| `/api/settings/database/backups/download` | `GET` | Admin | Baixa backup verificado com proteção `ResolveBackupPath`. |
| `/api/settings/database/restore` | `POST` | Admin | Restaura backup (`PRAGMA quick_check` / `pg_restore`). |

---

## 4. Saúde e Observabilidade Prometheus

* **`GET /healthz`**: Sonda leve de vivacidade retornando `200 OK` (`text/plain`).
* **`GET /api/health`**: Diagnóstico JSON completo informando status/latência do banco de dados, conexão MQTT, memória, goroutines e dispositivos.
* **`GET /metrics`**: Endpoint nativo no formato Prometheus:
  * `noxfort_uptime_seconds`
  * `noxfort_goroutines`
  * `noxfort_memory_heap_alloc_bytes`, `noxfort_memory_heap_sys_bytes`, `noxfort_memory_num_gc`
  * `noxfort_db_status`, `noxfort_db_latency_ms`
  * `noxfort_mqtt_connected`
  * `noxfort_devices_total`, `noxfort_devices_by_status`
  * `noxfort_telemetry_buffer_depth`
  * `noxfort_telemetry_ingested_total`, `noxfort_telemetry_rejected_total`
  * `noxfort_alerts_dispatched_total`
