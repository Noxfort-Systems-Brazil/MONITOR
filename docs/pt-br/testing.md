# 🧪 Testes e Garantia de Qualidade (QA)

Este documento especifica os fluxos de validação do **Noxfort Monitor™ v2.0**, incluindo testes unitários automatizados com mocks de repositório e verificação manual ponta a ponta (E2E) via MQTT e HTTP REST.

⬅️ [Central de Documentação](../README.md) | 🏛️ [Arquitetura](architecture.md) | 📡 [API](api_reference.md)

---

## 1. Suíte de Testes Automatizados

Para executar os testes do sistema completo (Backend em Go + Frontend em Vitest):
```bash
# Executa todos os testes (Go + Vitest)
make test

# Executa testes unitários do Backend Go (com detecção de concorrência)
make test-backend

# Executa testes do Frontend JavaScript (Vitest + Happy-DOM)
make test-frontend

# Executa análise estática de código (linter)
make lint
```

### 1.1 Pipeline de CI/CD (GitHub Actions)
Todos os testes e verificações estáticas são executados automaticamente a cada `push` e `pull request` no workflow `.github/workflows/ci.yml`.

### 1.2 Arquitetura de Mocks em Memória
A lógica de negócio em `internal/monitor` e `internal/security` independe de bancos em disco através de interfaces mock:
* `MockDeviceRepository`: Simula consulta de dispositivos e atualização de `LastSeen`.
* `MockTelemetryRepository`: Valida persistência de incidentes.
* `MockAlertDispatcher`: Confirma que alertas chegam à política de roteamento.
* `MockNotificationChannel`: Testa envio de canais sem realizar chamadas de rede reais para SMTP ou Telegram.

---

## 2. Validação Manual Ponta a Ponta (E2E)

### 2.1 Ingestão via MQTT (`mosquitto_pub`)
Com o broker protegido ativo (`make broker-auth && make broker-start`):

```bash
# Batimento Cardíaco (Keep-Alive)
mosquitto_pub -u "${MQTT_USER:-noxfort_user}" -P "${MQTT_PASSWORD:-noxfort_secret_pass_2026}" \
  -t "noxfort/devices/pump-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "INFO",
  "message": "System OK",
  "occurred_at": "2026-09-27T18:00:00Z"
}'

# Incidente Crítico (Dispara Alertas Imediatos)
mosquitto_pub -u "${MQTT_USER:-noxfort_user}" -P "${MQTT_PASSWORD:-noxfort_secret_pass_2026}" \
  -t "noxfort/devices/pump-01/telemetry" -m '{
  "category": "HARDWARE",
  "origin": "pump-01",
  "level": "CRITICAL",
  "message": "Alarme de Sobrepressão Hidráulica: 180 bar",
  "occurred_at": "2026-09-27T18:05:00Z"
}'
```

### 2.2 Ingestão via HTTP REST (`cURL`)

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "carina-core",
    "level": "WARNING",
    "message": "Latência de inferência 4.2ms (alvo: 3.5ms)",
    "occurred_at": "2026-09-27T18:10:00Z"
  }'
```
*Resposta esperada*: Código HTTP `200 OK` com payload `{"status":"received"}`.

### 2.3 Verificação de Observabilidade e Métricas

```bash
# 1. Checagem rápida de liveness
curl -i http://localhost:22100/healthz

# 2. Diagnóstico completo de saúde e latência
curl -s http://localhost:22100/api/health

# 3. Exportador de métricas Prometheus
curl -s http://localhost:22100/metrics
```

### 2.4 Validação de Backup do Banco de Dados

```bash
# 1. Executa backup a quente
make backup

# 2. Lista os backups existentes
make backup-list

# 3. Valida integridade do snapshot SQLite
sqlite3 backups/monitor_sqlite_*.db "PRAGMA integrity_check;"
```
