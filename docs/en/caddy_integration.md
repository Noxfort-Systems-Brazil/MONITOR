# Noxfort Monitor — Caddy Reverse Proxy Integration

Deploying Noxfort Monitor behind Caddy (`caddy/Caddyfile`) provides automatic TLS certificates (ACME / Let's Encrypt), clean HTTPS termination for browser operators and edge telemetry devices, and verified proxy header forwarding.

---

## 1. Proxy Header Forwarding & Trusted Proxies

In `caddy/Caddyfile`, Caddy forwards `X-Real-IP`, `X-Forwarded-For`, and `X-Forwarded-Proto` upstream to Noxfort Monitor.

To honor those headers safely without allowing header spoofing from untrusted clients:

1. Configure `TRUSTED_PROXIES` in your environment (e.g., `TRUSTED_PROXIES=127.0.0.1/32,172.16.0.0/12`).
2. Set `MONITOR_USE_HTTPS=true` when serving externally behind TLS.

When configured:
* `ExtractClientIP` attributes failed logins, `LoginAttemptTracker` lockouts, and `SecurityAuditLog` entries (`ip_address`) to the real downstream client IP.
* `IsHTTPSRequest` detects HTTPS termination and issues cookies with the `Secure` attribute.

---

## 2. Example `Caddyfile` Site Block

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

## 3. Testing Edge Ingestion Through Caddy (HTTPS)

1. **Request Zero-Trust Pairing**:

```bash
curl -X POST https://monitor.example.com/api/pair \
  -H "Content-Type: application/json" \
  -d '{"origin":"edge-01","name":"Boiler Sensor 01"}'
```

Expected response: `202 Accepted` (without an API key). An `ADMIN` opens `/devices`, approves the pending device, and copies the temporary 5-minute TTL `nox_live_` token.

2. **Publish Authenticated Telemetry**:

```bash
curl -X POST https://monitor.example.com/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_your_copied_token" \
  -H "X-Device-Origin: edge-01" \
  -d '{"category":"HARDWARE","origin":"edge-01","level":"INFO","message":"System OK (temp 41.2C)","occurred_at":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}'
```
