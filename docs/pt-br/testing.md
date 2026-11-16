# Noxfort Monitor — Guia de Testes e Verificação

O Noxfort Monitor valida o backend em Go, os scripts de interface frontend e os fluxos MQTT/HTTP ao vivo utilizando alvos padronizados no `Makefile`.

---

## 1. Suítes Automatizadas de Teste

Execute a suíte completa (backend Go + frontend Vitest):

```bash
make test
```

* **Backend (Go)**: Executa `go test ./... -v` cobrindo segurança (`lockout_test.go`, PBKDF2, complexidade de senha), persistência (`session_repo_test.go`, `retention_test.go`, migração transacional, backup/restore), middleware RBAC HTTP, política MQTT estrita, decodificador de protocolo e alertas (`alerts_test.go`).
* **Frontend (Vitest)**: Executa `npm test` (`make test-frontend`), rodando a suíte unitária em JavaScript via Vitest (4 arquivos de teste, 26 testes — substituindo o antigo uso de pytest).
* **Análise Estática**: Execute `go vet ./...` ou `make lint` para validar a integridade do código Go.

---

## 2. Teste Ao Vivo via MQTT (`mosquitto_pub`)

Como `MONITOR_MQTT_STRICT` vem habilitado como `true` por padrão, pré-cadastre `pump-01` como dispositivo `ACTIVE` em `/devices` (ou defina `MONITOR_MQTT_STRICT=false` em desenvolvimento local) antes de publicar em `noxfort/telemetry/#`:

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "noxfort/telemetry/pump-01" \
  -m '{"category":"HARDWARE","origin":"pump-01","level":"INFO","message":"System OK","occurred_at":"2026-09-27T18:00:00Z"}'
```

---

## 3. Teste Ao Vivo HTTP Zero-Trust (`curl`)

1. **Pré-registrar dispositivo (`POST /api/pair`)**:

```bash
curl -i -X POST http://127.0.0.1:22100/api/pair \
  -H "Content-Type: application/json" \
  -d '{"origin":"pump-02","name":"Bomba Hidráulica 02"}'
```

Resposta esperada: `202 Accepted` sem chave na resposta. Acesse `/devices` como `ADMIN`, aprove `pump-02` e copie a chave temporária `nox_live_` (TTL de 5 minutos).

2. **Enviar telemetria autenticada (`POST /api/telemetry`)**:

```bash
curl -i -X POST http://127.0.0.1:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_SUA_CHAVE_COPIADA" \
  -H "X-Device-Origin: pump-02" \
  -d '{"category":"HARDWARE","origin":"pump-02","level":"WARNING","message":"Temperatura subindo (39.5C)","occurred_at":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}'
```

Resposta esperada: `202 Accepted`, atualizando `GET /api/dashboard/summary` e os contadores de `GET /metrics`.

---

## 4. Modo Demonstração e Apresentação Automatizada

Execute `make record-demo` (`scripts/demo/record_presentation.cjs` e `scripts/demo/mock_telemetry.cjs`) contra `--demo` (`RunDesktopServer()`) para validar fluxos de interface com telemetria simulada.
