# Noxfort Monitor — Guia de Implantação e Operação

Este guia apresenta as variáveis de ambiente atuais, orquestração com Docker Compose, configuração de proxy reverso, backups automatizados e métricas de produção.

---

## 1. Configuração por Ambiente (`.env`)

O arquivo legado `configs/config.yaml` foi removido. Variáveis de inicialização são carregadas do `.env` via `LoadDotEnv()` (consulte `.env.example`), enquanto parâmetros operacionais dinâmicos (URL do broker MQTT, alertas, túneis) ficam persistidos na tabela SQL `settings`.

| Variável | Padrão | Descrição |
| :--- | :--- | :--- |
| `MONITOR_MQTT_STRICT` | `true` | Descarta telemetria MQTT (`noxfort/telemetry/#`) de dispositivos não cadastrados ou `PAUSED`. |
| `TRUSTED_PROXIES` | *(vazio)* | Lista separada por vírgulas de CIDRs/IPs confiáveis para cabeçalhos `X-Real-IP`, `X-Forwarded-For` e `X-Forwarded-Proto`. |
| `MONITOR_USE_HTTPS` | `false` | Habilita cookies `Secure` e detecção HTTPS atrás de terminadores TLS. |
| `MONITOR_RETENTION_DAYS` | `90` | Prazo em dias para o `RetentionManager` limpar `telemetry`, logs de auditoria, alertas e transições (a cada 24h). |
| `MONITOR_BACKUP_INTERVAL_HOURS` | `24` | Intervalo de backups quentes agendados pelo `BackupManager` (retém até `7` arquivos). |
| `MONITOR_DB_DRIVER` / `MONITOR_DB_*` | `sqlite` | Configuração opcional de bootstrap direto com PostgreSQL. |

---

## 2. Docker Compose e Modo Headless

```bash
cp .env.example .env
docker compose up -d --build
```

* Executa o binário com `--headless` junto ao Eclipse Mosquitto (`mosquitto/config/mosquitto.conf`) e PostgreSQL opcional (`docker-compose.yml`).
* Execute `./scripts/setup_mqtt_auth.sh` para configurar autenticação no Mosquitto antes de expor a porta MQTT em rede.

---

## 3. Exemplo de Proxy Reverso NGINX

Ao usar NGINX em vez de `caddy/Caddyfile`, garanta o repasse dos cabeçalhos de proxy e inclua o IP do NGINX em `TRUSTED_PROXIES`:

```nginx
location / {
    proxy_pass http://127.0.0.1:22100;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

---

## 4. Backups, Restauração, Retenção e Métricas

* **Backups Agendados e Sob Demanda**: `BackupManager` cria snapshots atômicos (`VACUUM INTO` no SQLite; `pg_dump -F c` no PostgreSQL), mantém até 7 cópias e oferece `/api/settings/database/backups/download` + `/api/settings/database/restore` (`PRAGMA quick_check` / `pg_restore`).
* **Retenção Automática**: `RetentionManager` limpa dados expirados a cada 24h em `telemetry`, `alert_dispatch_logs`, `security_audit_logs` e `device_state_transitions`.
* **Coleta Prometheus (`GET /metrics`)**: Expõe uptime, goroutines, heap, `noxfort_db_status`, `noxfort_db_latency_ms`, `noxfort_mqtt_connected`, `noxfort_devices_total`, `noxfort_devices_by_status`, `noxfort_telemetry_buffer_depth`, `noxfort_telemetry_ingested_total`, `noxfort_telemetry_rejected_total` e `noxfort_alerts_dispatched_total`.
