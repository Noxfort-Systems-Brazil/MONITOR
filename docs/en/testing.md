# Noxfort Monitor — Testing & Verification Guide

Noxfort Monitor validates backend Go services, frontend UI scripts, and live MQTT/HTTP edge pipelines using unified `Makefile` targets and standard toolchains.

---

## 1. Automated Test Suites

Run the complete test suite (Go backend + Vitest frontend):

```bash
make test
```

* **Backend (Go)**: Runs `go test ./... -v` across security (`lockout_test.go`, PBKDF2 hashing, complexity), storage (`session_repo_test.go`, `retention_test.go`, transactional migration, backups/restore), HTTP RBAC/ingestion, MQTT strict policy, protocol decoding, and monitor alerts (`alerts_test.go`).
* **Frontend (Vitest)**: Runs `npm test` (`make test-frontend`), executing the JavaScript unit test suite (4 test files, 26 tests) via Vitest (replaces legacy pytest tooling).
* **Static Analysis**: Run `go vet ./...` or `make lint` to verify Go code hygiene.

---

## 2. Live MQTT Testing (`mosquitto_pub`)

Because `MONITOR_MQTT_STRICT` defaults to `true`, pre-register `pump-01` as an `ACTIVE` device in `/devices` (or set `MONITOR_MQTT_STRICT=false` in local development) before publishing to the `noxfort/telemetry/#` hierarchy:

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "noxfort/telemetry/pump-01" \
  -m '{"category":"HARDWARE","origin":"pump-01","level":"INFO","message":"System OK","occurred_at":"2026-09-27T18:00:00Z"}'
```

---

## 3. Live Zero-Trust HTTP Testing (`curl`)

1. **Pre-register device (`POST /api/pair`)**:

```bash
curl -i -X POST http://127.0.0.1:22100/api/pair \
  -H "Content-Type: application/json" \
  -d '{"origin":"pump-02","name":"Hydraulic Pump 02"}'
```

Expected: `202 Accepted` without a key. Open `/devices` as `ADMIN`, approve `pump-02`, and copy its 5-minute TTL `nox_live_` token.

2. **Ingest telemetry (`POST /api/telemetry`)**:

```bash
curl -i -X POST http://127.0.0.1:22100/api/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Device-Token: nox_live_YOUR_COPIED_KEY" \
  -H "X-Device-Origin: pump-02" \
  -d '{"category":"HARDWARE","origin":"pump-02","level":"WARNING","message":"Temperature rising (temp 39.5C)","occurred_at":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}'
```

Expected: `202 Accepted`, followed by visible live state updates on `GET /api/dashboard/summary` and counter increments on `GET /metrics`.

---

## 4. Automated Presentation & Demo Mode

Run `make record-demo` (`scripts/demo/record_presentation.cjs` and `scripts/demo/mock_telemetry.cjs`) against `--demo` (`RunDesktopServer()`) to exercise UI workflows and simulated telemetry end-to-end.
