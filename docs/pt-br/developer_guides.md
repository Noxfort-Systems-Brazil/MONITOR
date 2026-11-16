# Noxfort Monitor — Guias de Desenvolvimento e Convenções

Este guia documenta as convenções do repositório, contratos de persistência, auxiliares HTTP/segurança e ciclo de vida em segundo plano.

---

## 1. Comandos do `Makefile`

* `make test`: Executa todos os testes Go (`go test ./... -v`) e frontend Vitest (`npm test`).
* `make test-frontend`: Executa apenas os testes Vitest em JavaScript (`npm test`).
* `make lint`: Executa `go vet` e checagens de qualidade.
* `make record-demo`: Executa os fluxos automatizados em `scripts/demo/` sobre o modo `--demo`.

---

## 2. Contrato `ReloadableRepository`

Todo repositório gerenciado pelo `DBManager` deve implementar:

```go
type ReloadableRepository interface {
    SetDB(db *sql.DB, driver string)
}
```

Atualmente 7 repositórios implementam essa interface (`Device`, `Telemetry`, `Settings`, `Contact`, `Audit`, `User` e `DBSessionStore`), permitindo que `Migrator.MigrateData` (em transação única sobre as 9 tabelas) alterne SQLite ↔ PostgreSQL sem perder sessões nem estado operacional.

---

## 3. Convenções de Handlers HTTP e Segurança

* **Renderização de Templates**: Utilize `RenderPageWithRequest` para manter navegação, contexto do usuário e cabeçalhos consistentes.
* **Respostas `403 Forbidden`**: Use `writeForbiddenResponse` para retornar JSON em APIs/XHR e `web/templates/forbidden.html` na UI HTML.
* **Tokens e IPs Confiáveis**: Use sempre `ExtractSessionToken` (que rejeita `?token=` na query string) e `ExtractClientIP` / `IsHTTPSRequest` (que validam `TRUSTED_PROXIES` e `MONITOR_USE_HTTPS`).
* **Redação de Segredos**: Nunca exponha `ApiKey` ativo ou `duckdns_token` em claro no HTML renderizado.

---

## 4. Métricas, Rastreabilidade de Alertas e Encerramento

* **Instrumentação (`internal/metrics`)**: Atualize os contadores e gauges usando `IncTelemetryIngested`, `IncTelemetryRejected`, `IncAlertDispatched` e `SetBufferDepth` para manter `/metrics` sincronizado.
* **Vínculo Telemetria ↔ Alerta**: Telemetria comum usa `BufferedTelemetryWriter`, enquanto transições detectadas pelo `StateManager`/`Engine` gravam com `SaveEventReturningID` e disparam `TriggerAlertWithTelemetryID` para preencher `alert_dispatch_logs.telemetry_id`.
* **Ciclo de Vida (`bgCtx`)**: Propague o contexto de cancelamento para `RetentionManager`, `BackupManager.StartScheduledBackups`, limpeza de sessões e loop MQTT, e finalize com `AlertService.Shutdown(shutdownCtx)` antes de fechar conexões SQL.
