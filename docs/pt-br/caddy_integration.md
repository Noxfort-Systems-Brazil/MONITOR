# Noxfort Monitor — Integração com Proxy Reverso Caddy

A implantação atrás do Caddy (`caddy/Caddyfile`) fornece certificados TLS automáticos (ACME / Let's Encrypt), terminação HTTPS transparente para operadores e dispositivos, e encaminhamento confiável de cabeçalhos de proxy.

---

## 1. Cabeçalhos Encaminhados e `TRUSTED_PROXIES`

No arquivo `caddy/Caddyfile`, o Caddy encaminha `X-Real-IP`, `X-Forwarded-For` e `X-Forwarded-Proto` para o servidor backend.

Para validar esses cabeçalhos sem abrir brechas para falsificação (*IP spoofing*):

1. Configure `TRUSTED_PROXIES` no `.env` (ex.: `TRUSTED_PROXIES=127.0.0.1/32,172.16.0.0/12`).
2. Defina `MONITOR_USE_HTTPS=true` quando exposto externamente via HTTPS.

Com isso:
* `ExtractClientIP` registra o IP real do cliente no `LoginAttemptTracker` e nos logs de `SecurityAuditLog` (`ip_address`).
* `IsHTTPSRequest` detecta conexões seguras e aplica o atributo `Secure` nos cookies de sessão.

---

## 2. Exemplo de `Caddyfile`

```caddyfile
monitor.example.com {
    reverse_proxy 127.0.0.1:22100 {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }
}
```

---

## 3. Testando Ingestão Zero-Trust via Caddy (HTTPS)

1. **Solicitar Pareamento (`POST /api/pair`)**:

```bash
curl -X POST https://monitor.example.com/api/pair \
  -H "Content-Type: application/json" \
  -d '{"origin":"edge-01","name":"Sensor Caldeira 01"}'
```

Resposta esperada: `202 Accepted` (sem chave na resposta). Um usuário `ADMIN` aprova o dispositivo em `/devices` e copia a chave temporária `nox_live_` (TTL de 5 minutos).

2. **Enviar Telemetria Autenticada (`POST /api/telemetry`)**:

```bash
curl -X POST https://monitor.example.com/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_sua_chave_copiada" \
  -H "X-Device-Origin: edge-01" \
  -d '{"category":"HARDWARE","origin":"edge-01","level":"INFO","message":"System OK (temp 41.2C)","occurred_at":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}'
```
