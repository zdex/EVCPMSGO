# CPMS Core (Go) — Starter Kit (Tokenization-ready boundaries)

This is a minimal CPMS Core that works with the Go OCPP Gateway.

## What it does (MVP)
- **Auth** endpoint for gateway WS handshake:
  - `POST /v1/gateway/chargers/{chargePointId}/auth`
- **Event ingestion** endpoint (gateway -> CPMS):
  - `POST /v1/gateway/events`
  - stores raw events + updates "current state" + sessions
- Basic read APIs:
  - `GET /v1/chargers/{chargePointId}`
  - `GET /v1/chargers/{chargePointId}/connectors`
  - `GET /v1/sessions/{sessionId}`
  - `GET /v1/chargers/{chargePointId}/sessions?limit=50`

## Quick start (with Postgres)
1) Start Postgres:
```bash
docker compose up -d db
```

2) Apply schema:
```bash
psql "postgres://cpms:cpms@localhost:5432/cpms?sslmode=disable" -f db/schema.sql
```

3) Seed a charger (CP-123/devsecret):
```bash
go run ./cmd/seed --id CP-123 --secret devsecret
```

4) Run CPMS:
```bash
export CPMS_LISTEN_ADDR=:8081
export CPMS_DATABASE_URL="postgres://cpms:cpms@localhost:5432/cpms?sslmode=disable"
go run ./cmd/cpms
```

5) Point your Gateway at this CPMS:
```bash
export CPMS_BASE_URL=http://localhost:8081
export REQUIRE_CPMS_AUTH=true
go run ./cmd/gateway
```

DB schema here is intentionally minimal; we can expand later for sites, tariffs, wallets, settlement, invoices.


## Send commands (CPMS -> Gateway -> Charger)
Set gateway base URL when running CPMS (defaults to http://localhost:8080):
```bash
export GATEWAY_BASE_URL=http://localhost:8080
export GATEWAY_API_KEY=
```

Example RemoteStart via CPMS:
```bash
curl -X POST http://localhost:8081/v1/commands \
  -H "Content-Type: application/json" \
  -d '{
    "type":"RemoteStartTransaction",
    "chargePointId":"CP-123",
    "idempotencyKey":"start-1",
    "payload":{"connectorId":1,"idTag":"APP_abc123"}
  }'
```


## Session finalization fallback
This CPMS finalizes sessions with this fallback order when `meterStopWh` is missing:
1) `meterStopWh - meterStartWh` (StopTransaction)
2) last `Energy.Active.Import.Register` from `MeterSample` minus `meterStartWh`
3) sum of `Energy.Active.Import.Interval` samples (Wh)
4) mark as `Missing` and `is_estimated=true`

Migration for existing DB:
```bash
docker exec -i <db_container> psql -U cpms -d cpms < db/003_session_finalization.sql
```
