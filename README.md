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


## Finalize a session (manual)
If MeterValues arrive late (after StopTransaction), you can re-run finalization:

```bash
curl -X POST http://localhost:8081/v1/sessions/<sessionId>/finalize
```

Force recompute:
```bash
curl -X POST "http://localhost:8081/v1/sessions/<sessionId>/finalize?force=true"
```


## Pricing (per kWh MVP)
This version supports **per-kWh pricing**. When a session is finalized (TransactionEnded), CPMS computes:

`cost = (energy_wh / 1000) * price_per_kwh`

### Setup (existing DB)
Run migrations (in order) if you upgraded from earlier versions:
```bash
docker exec -i <db_container> psql -U cpms -d cpms < db/003_session_finalization.sql
docker exec -i <db_container> psql -U cpms -d cpms < db/004_pricing.sql
```

### Create site + active tariff (API)
```bash
curl -X POST http://localhost:8081/v1/sites -H "Content-Type: application/json" -d '{"name":"DemoSite"}'
# use returned siteId
curl -X POST http://localhost:8081/v1/sites/<siteId>/tariffs -H "Content-Type: application/json" -d '{"pricePerKwh":0.30,"currency":"USD"}'
```

### Assign charger to site (DB quick)
```bash
docker exec -it <db_container> psql -U cpms -d cpms -c "update chargers set site_id='<siteId>' where charge_point_id='CP-123';"
```

After the next finalized session, `tariffId`, `costAmount`, `costCurrency`, `pricedAt` will appear on the session.


## Settlement layer (tokenization-ready)
After a session is **finalized + priced**, CPMS creates **one Pending settlement** (idempotent, 1 per session).
This is the clean boundary for tokenization/minting later.

### Migration
```bash
docker exec -i <db_container> psql -U cpms -d cpms < db/005_settlements.sql
```

### Set a payout wallet for a site
```bash
curl -X POST http://localhost:8081/v1/sites/<siteId>/wallet \
  -H "Content-Type: application/json" \
  -d '{"wallet":"rEXAMPLE_XRPL_ADDRESS"}'
```

### List pending settlements
```bash
curl "http://localhost:8081/v1/settlements?status=Pending&limit=50"
```

### Mark as submitted (after you broadcast a chain tx)
```bash
curl -X POST http://localhost:8081/v1/settlements/<settlementId>/submitted \
  -H "Content-Type: application/json" \
  -d '{"chain":"XRPL","txHash":"ABC123..."}'
```

### Mark confirmed / failed
```bash
curl -X POST http://localhost:8081/v1/settlements/<settlementId>/confirmed
curl -X POST http://localhost:8081/v1/settlements/<settlementId>/failed -H "Content-Type: application/json" -d '{"error":"insufficient fee"}'
```
