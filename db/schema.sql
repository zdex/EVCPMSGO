create extension if not exists "uuid-ossp";

create table if not exists chargers (
  charge_point_id text primary key,
  secret_hash text not null default '',
  is_active boolean not null default false,
  vendor text,
  model text,
  ocpp_version text not null default '1.6J',
  last_seen_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists gateway_events (
  id bigserial primary key,
  charge_point_id text not null references chargers(charge_point_id) on delete cascade,
  event_type text not null,
  ts timestamptz not null,
  payload jsonb not null,
  received_at timestamptz not null default now()
);
create index if not exists idx_gateway_events_cp_ts on gateway_events(charge_point_id, ts);

create table if not exists connector_state (
  charge_point_id text not null references chargers(charge_point_id) on delete cascade,
  connector_id int not null,
  status text not null,
  error_code text not null default 'NoError',
  updated_at timestamptz not null default now(),
  primary key (charge_point_id, connector_id)
);

create table if not exists sessions (
  session_id uuid primary key default uuid_generate_v4(),
  charge_point_id text not null references chargers(charge_point_id) on delete cascade,
  connector_id int not null,
  transaction_id int not null,
  id_tag text,
  started_at timestamptz not null,
  ended_at timestamptz,
  meter_start_wh bigint,
  meter_stop_wh bigint,
  reason text,
  energy_wh bigint,
  energy_source text,
  is_estimated boolean not null default false,
  finalized_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index if not exists idx_sessions_cp_started on sessions(charge_point_id, started_at desc);
create index if not exists idx_sessions_cp_tx on sessions(charge_point_id, transaction_id);

create table if not exists meter_samples (
  id bigserial primary key,
  session_id uuid not null references sessions(session_id) on delete cascade,
  charge_point_id text not null references chargers(charge_point_id) on delete cascade,
  transaction_id int not null,
  ts timestamptz not null,
  samples_json jsonb not null
);
create index if not exists idx_meter_samples_session_ts on meter_samples(session_id, ts);

create table if not exists commands (
  command_id uuid primary key default uuid_generate_v4(),
  charge_point_id text not null references chargers(charge_point_id) on delete cascade,
  type text not null,
  idempotency_key text not null unique,
  payload jsonb not null,
  status text not null default 'Queued',
  response jsonb,
  error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index if not exists idx_commands_cp_created on commands(charge_point_id, created_at desc);


create table if not exists sites (
  site_id uuid primary key default uuid_generate_v4(),
  name text not null unique,
  created_at timestamptz not null default now()
);

alter table chargers
  add column if not exists site_id uuid references sites(site_id);

create table if not exists tariffs (
  tariff_id uuid primary key default uuid_generate_v4(),
  site_id uuid not null references sites(site_id) on delete cascade,
  price_per_kwh numeric(12,4) not null,
  currency text not null default 'USD',
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index if not exists idx_tariffs_site_active on tariffs(site_id, is_active);

alter table sessions
  add column if not exists tariff_id uuid references tariffs(tariff_id),
  add column if not exists cost_amount numeric(12,4),
  add column if not exists cost_currency text,
  add column if not exists priced_at timestamptz;

create index if not exists idx_sessions_priced_at on sessions(priced_at);


alter table sites
  add column if not exists payout_wallet text;

create table if not exists settlements (
  settlement_id uuid primary key default uuid_generate_v4(),
  session_id uuid not null references sessions(session_id) on delete cascade,
  site_id uuid not null references sites(site_id) on delete cascade,
  amount numeric(12,4) not null,
  currency text not null,
  status text not null default 'Pending', -- Pending|Submitted|Confirmed|Failed
  chain text,              -- e.g., XRPL, ETH, SOL (optional)
  tx_hash text,            -- blockchain tx hash once submitted
  external_ref text,       -- idempotency / external correlation key
  error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique(session_id)
);
create index if not exists idx_settlements_status_created on settlements(status, created_at);
