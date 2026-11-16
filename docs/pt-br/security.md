# Noxfort Monitor — Arquitetura de Segurança

O Noxfort Monitor aplica defesa em profundidade abrangendo controle de acesso, derivação de credenciais, persistência de sessões, proteção contra força bruta, pareamento Zero-Trust e canais de alerta.

---

## 1. Controle de Acesso Baseado em Perfis (RBAC)

Dois perfis controlam o acesso na API HTTP e na interface desktop (Wails):

* **`ADMIN`**: Controle total sobre dispositivos/chaves, gestão de usuários/perfis/desbloqueios, contatos, configurações do sistema/MQTT, túneis DuckDNS/Ngrok e migração/backup/restauração de banco de dados. Contas superuser e último administrador são protegidos contra exclusão (`DeleteUserByActor`).
* **`OPERATOR` (Estritamente Somente Leitura)**: Acesso limitado à visualização (`GET` dashboard, `/audit`, `/api/audit/*`, `/remote`, `/server`, `/settings`), `/api/auth/logout` e autoatendimento `POST /api/users/password` (exige `current_password`). Bloqueado de `/users`, mutações em `/api/users`, `/api/settings/database/backup*` e qualquer alteração de estado:
  * Requisições API/XHR (`/api/*`, `Accept: application/json`, `XMLHttpRequest`) recebem JSON `403 Forbidden`.
  * Views HTML do desktop/navegador exibem a página `web/templates/forbidden.html`.

---

## 2. Hashing de Senhas e Complexidade

* **Derivação PBKDF2**: Senhas são processadas pelo `PBKDF2Hasher` via **PBKDF2 HMAC-SHA256** com **`100.000` iterações**, sal aleatório de **`16` bytes** e chave derivada de **`32` bytes**.
* **Política de Complexidade**: `ValidatePasswordComplexity` valida comprimento mínimo e diversidade de caracteres no cadastro, troca de senha e inicialização de superusuário.

---

## 3. Proteção contra Força Bruta (`LoginAttemptTracker`)

* Falhas de login são contabilizadas por chave composta `username|ip` (IP resolvido por `ExtractClientIP`).
* **Regra de Bloqueio**: **`5` tentativas inválidas** dentro de **`15` minutos** acionam bloqueio de **`5` minutos**, respondendo com HTTP **`429 Too Many Requests`** e registrando `AUTH_LOGIN_LOCKED`.
* Administradores podem liberar bloqueios manualmente via `POST /api/users/unlock`.

---

## 4. Sessões Persistentes Hasheadas e Proxies Confiáveis

* **`DBSessionStore`**: Sessões são armazenadas na tabela `sessions`, guardando exclusivamente o hash **`SHA-256` (`token_hash`)** do token, com **TTL fixo de `24h`** e limpeza automática a cada hora. Ao implementar `ReloadableRepository`, sessões permanecem ativas após trocas ao vivo entre SQLite e PostgreSQL.
* **Proibição de `?token=` na URL**: `ExtractSessionToken` lê apenas cookies HTTP-only e rejeita tokens na query string (`?token=`) para evitar vazamento em logs e histórico.
* **`TRUSTED_PROXIES` e `MONITOR_USE_HTTPS`**: `ExtractClientIP` e `IsHTTPSRequest` só confiam nos cabeçalhos `X-Real-IP`, `X-Forwarded-For` e `X-Forwarded-Proto` (enviados pelo `caddy/Caddyfile`) quando originados de proxies em `TRUSTED_PROXIES`, ativando cookies `Secure` corretamente sem permitir falsificação de IP.

---

## 5. Proteção de Borda, Redação de Segredos e SMTP

* **Pareamento Zero-Trust**: `POST /api/pair` aceita pré-cadastro como `PENDING` (limite de 50 na fila, `409 Conflict` se já `ACTIVE`) retornando `202` sem credenciais; a chave `nox_live_` (TTL 5m) é copiada fora de banda em `/devices` e validada em tempo constante junto a `X-Device-Origin` e janela de replay `±300s`.
* **MQTT Estrito**: `MONITOR_MQTT_STRICT` (padrão `true`) descarta publicações em `noxfort/telemetry/#` de dispositivos não registrados ou pausados.
* **Mascaramento em HTML**: `/settings`, `/remote` e `/devices` ocultam segredos ativos (`ApiKey`, `duckdns_token`) no HTML renderizado; salvamentos parciais preservam tokens não alterados.
* **Endurecimento SMTP**: `EmailChannel` suporta TLS implícito na porta `465` (`TLS 1.2+`) e sanitiza cabeçalhos (`sanitizeEmailHeader`, removendo CRLF) contra injeção SMTP.
