# 📡 Referência de API e Protocolos de Ingestão

Este documento especifica formalmente as interfaces de comunicação do **Noxfort Monitor™ v2.0**, cobrindo o protocolo de ingestão assíncrono via MQTT, o endpoint HTTP REST de telemetria e as APIs administrativas.

⬅️ [Central de Documentação](../README.md) | 🏛️ [Arquitetura](architecture.md) | 🗄️ [Banco de Dados](database.md) | 🔐 [Segurança](security.md)

---

## 1. Protocolo de Ingestão MQTT

* **Endereço do Broker**: `tcp://127.0.0.1:1883`
* **Padrão de Tópico Padrão**: `noxfort/devices/{identifier}/telemetry`
* **Subscrição com Curinga**: `noxfort/devices/+/telemetry`

### Formato JSON Universal `IncomingEvent`

```json
{
  "category": "HARDWARE",
  "origin": "sensor-node-tx1",
  "level": "CRITICAL",
  "message": "Superaquecimento detectado: Temperatura 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

#### Especificação dos Campos:
* **`category`** (*String*): `HARDWARE` ou `SOFTWARE`. Controla as regras de roteamento RBAC (Hardware $\rightarrow$ Técnicos; Software $\rightarrow$ Programadores).
* **`origin`** (*String*): Identificador exclusivo do nó (ex: `carina`, `synapse`, `pump-01`).
* **`level`** (*String*): `INFO`, `WARNING` ou `CRITICAL`.
  * Mensagens `INFO` contendo palavras-chave de keep-alive ("*system ok*", "*heartbeat*", "*online*") atualizam o timestamp de presença sem gerar ruído de alertas nem gravações desnecessárias no banco de dados.
  * Mensagens `CRITICAL` disparam envio imediato de alertas multicanal.
* **`message`** (*String*): Descrição legível da ocorrência operacional.
* **`occurred_at`** (*String ISO-8601*): Timestamp UTC.

---

## 2. Ingestão de Telemetria via HTTP REST

### `POST /api/telemetry`
Projetado para agentes de campo (**Carina**, **Synapse**, microcontroladores IoT ou scripts cURL).

* **Porta**: `22100` (ou URL pública HTTPS via DuckDNS/Caddy ou Ngrok)
* **Autenticação**: Pública (isenta de middleware para atender nós autônomos de campo).
* **Restrição de Rede**: No servidor externo (`ExternalIngestionHandler`), `POST /api/telemetry`, `/healthz`, `/api/health` e `/metrics` são acessíveis na porta `22100`. Requisições web/navegador a rotas de dashboard retornam **403 Forbidden**.
* **Cabeçalho**: `Content-Type: application/json`
* **Corpo**: Idêntico ao esquema JSON `IncomingEvent`.

```bash
curl -X POST http://localhost:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Pool de conexões em 85% da capacidade.",
    "occurred_at": "2026-09-27T18:00:00Z"
  }'
```

---

## 3. Endpoints Administrativos e de Operação

*Disponíveis dentro da aplicação desktop nativa ou através de sessões autenticadas.*

| Endpoint | Método | Papel Mínimo | Descrição |
|---|---|---|---|
| `/api/auth/login` | `POST` | Público | Autentica credenciais e emite cookie `noxfort_session`. |
| `/api/auth/logout` | `POST` | Autenticado | Invalida o token de sessão ativo. |
| `/api/auth/status` | `GET` | Público | Retorna estado de autenticação, usuário e papel da sessão. |
| `/api/auth/register` | `POST` | Admin | Alias protegido de criação de conta com validação de complexidade de senha. |
| `/devices` | `GET` | Operador+ | Renderiza a tela de gerenciamento de sistemas com equipamentos e `last_seen`. |
| `/devices/delete` | `POST` | Admin | Remove um dispositivo via formulário POST (`id=xyz`) e redireciona (303). |
| `/contacts` | `GET` | Operador+ | Renderiza o painel de gerenciamento da Equipe de Resposta. |
| `/contacts/create` | `POST` | Admin | Cadastra novo contato de alerta (nome, email, telefone, telegram_chat_id, cargo). |
| `/contacts/update` | `POST` | Admin | Atualiza dados e canais de notificação de um contato existente. |
| `/contacts/delete` | `POST` | Admin | Remove um contato da equipe de resposta. |
| `/api/users` | `GET` | Admin | Retorna listagem JSON com todos os operadores cadastrados. |
| `/api/users/create` | `POST` | Admin | Cria nova conta de operador ou admin (exige senha forte de 8+ dígitos). |
| `/api/users/delete` | `POST` | Admin | Remove uma conta de operador (o superusuário é protegido contra exclusão). |
| `/api/settings/database/status` | `GET` | Autenticado | Retorna tipo do banco ativo (Postgres/SQLite), versão e latência. |
| `/api/settings/database/test` | `POST` | Admin | Testa credenciais do PostgreSQL antes da troca. |
| `/api/settings/database/save` | `POST` | Admin | Recarrega repositórios ativos com o motor de destino (opção `migrate=true`). |
| `/api/settings/database/provision-user` | `POST` | Admin | Cria usuário dedicado com privilégios restritos no esquema do PostgreSQL. |
| `/api/settings/database/backup` | `POST` | Admin | Dispara um snapshot consistente a quente do banco de dados (`VACUUM INTO` / `pg_dump`). |
| `/api/settings/database/backups` | `GET` | Admin | Lista os snapshots de backup salvos no diretório `backups/`. |
| `/api/tunnel/status` | `GET` | Autenticado | Retorna estado, provedor (DuckDNS/Ngrok), URLs e IPv6. |
| `/api/tunnel/save` | `POST` | Admin | Persiste credenciais do DuckDNS / Ngrok no banco. |
| `/api/tunnel/start` | `POST` | Admin | Dispara atualização dinâmica DuckDNS ou abre túnel. |
| `/api/tunnel/stop` | `POST` | Admin | Pausa atualizações e desativa inicialização automática. |
| `/api/tunnel/disconnect` | `POST` | Admin | Desvincula credenciais salvas e encerra o serviço. |
| `/api/tunnel/test` | `POST` | Admin | Valida ativamente token e resolução DNS do DuckDNS. |
| `/api/audit/security` | `GET` | Admin | Consulta logs de segurança e tentativas de login. |
| `/api/audit/alerts` | `GET` | Admin | Consulta logs de SLA e entrega de alertas. |
| `/api/audit/transitions` | `GET` | Admin | Consulta transições de estado dos nós (quedas/recuperações). |
| `/settings/test` | `POST` | Admin | Dispara e-mail de teste utilizando as configurações SMTP ativas. |
| `/settings/test-telegram` | `POST` | Admin | Dispara mensagem de teste MarkdownV2 para o Telegram configurado. |
| `/api/open-external` | `POST` | Autenticado | Solicita a abertura de URL externa no navegador padrão do sistema operacional. |
| `/api/window/toggle-fullscreen` | `POST` | Público | Alterna a janela Wails entre modo tela cheia e normal. |
| `/api/window/exit-fullscreen` | `POST` | Público | Sai do modo tela cheia. |

---

## 4. Endpoints Públicos de Observabilidade e Diagnóstico

*Isentos de middleware para permitir scraping pelo Prometheus e checagens de liveness em contêineres.*

| Endpoint | Método | Descrição | Formato |
|---|---|---|---|
| `/healthz` | `GET` | Probe de liveness para orquestradores (retorna 200 `OK`). | `text/plain` |
| `/api/health` | `GET` | Diagnóstico de saúde detalhado: estado e latência do banco de dados, status MQTT, uso de memória RAM, goroutines e dispositivos cadastrados. | `application/json` |
| `/metrics` | `GET` | Métricas no formato do **Prometheus** (`noxfort_uptime_seconds`, `noxfort_goroutines`, `noxfort_db_status`, `noxfort_mqtt_connected`, etc.). | `text/plain; version=0.0.4` |
