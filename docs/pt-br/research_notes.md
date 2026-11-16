# Noxfort Monitor — Registros de Decisão Arquitetural (ADRs)

Este documento registra as principais decisões de engenharia, compromisso de segurança e evolução estrutural do Noxfort Monitor.

---

## ADR-01: Remoção de `configs/config.yaml` em Favor de `.env` + Tabela `settings`

* **Contexto**: Manter parâmetros duplicados entre arquivos YAML e banco SQL gerava inconsistência entre execução desktop e Docker.
* **Decisão**: O arquivo `configs/config.yaml` foi removido. Variáveis de bootstrap carregam via `LoadDotEnv()` a partir do `.env`, enquanto configurações dinâmicas (MQTT broker URL, alertas, túneis) ficam no SQL `settings` com gatilho imediato (`SetOnMQTTConfigChanged` -> `Reconnect(brokerURL)`).

---

## ADR-02: Pareamento Zero-Trust Fora de Banda (`POST /api/pair`) e MQTT Estrito

* **Contexto**: Auto-registrar dispositivos desconhecidos ou devolver chaves na resposta do `/api/pair` abria vetor de *spoofing* e saturação de fila.
* **Decisão**:
  * `POST /api/pair` limita payload a 64 KB, limita a fila `PENDING` a 50 entradas (`429`), retorna `409 Conflict` se já `ACTIVE` e responde `202 Accepted` **sem chave de API**.
  * Um `ADMIN` aprova em `/devices` e copia a chave `nox_live_` (TTL de 5 minutos); `POST /api/telemetry` exige `X-Device-Token` / `Bearer`, `X-Device-Origin` e janela de replay `±300s`.
  * `MONITOR_MQTT_STRICT=true` descarta publicações de dispositivos desconhecidos ou `PAUSED` em `noxfort/telemetry/#`.

---

## ADR-03: Sessões Persistentes Hasheadas (`DBSessionStore`) e Bloqueio Anti-Brute-Force

* **Contexto**: Mapas em memória perdiam sessões em reinicializações ou trocas de banco, e faltava limitação de tentativas de login.
* **Decisão**:
  * `DBSessionStore` persiste hashes `SHA-256` (`token_hash`) na tabela `sessions` (TTL de 24h, limpeza horária) implementando `ReloadableRepository`.
  * `PBKDF2Hasher` (`100.000` iterações HMAC-SHA256) e `LoginAttemptTracker` (5 falhas em 15m -> bloqueio de 5m com HTTP `429`) protegem a autenticação.
  * `?token=` na URL foi explicitamente bloqueado e o perfil `OPERATOR` foi travado como estritamente somente leitura (+ autoatendimento `POST /api/users/password`).

---

## ADR-04: Integridade Dual-Engine (SQLite WAL), Backup/Restauração e Retenção

* **Decisão**:
  * Habilitado SQLite em modo WAL (`journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout=5000`).
  * `Migrator.MigrateData` executa em **transação única** nas 9 tabelas, abortando `DBManager.Switch` em caso de falha.
  * Introduzidos `BackupManager` (`VACUUM INTO`, `pg_dump -F c`, download + `PRAGMA quick_check` / `pg_restore`) e `RetentionManager` (`MONITOR_RETENTION_DAYS=90`).

---

## ADR-05: Métricas Prometheus Nativas e Vínculo Síncrono de Alertas

* **Decisão**: Criado `internal/metrics` (`/metrics`), adicionadas retentativas com backoff (`500ms`, `1500ms`) + `AlertService.Shutdown(ctx)` e persistência síncrona `SaveEventReturningID` em transições para vincular `alert_dispatch_logs.telemetry_id`.
