-- Migration: settlement layer (tokenization-ready)
alter table sites
  add column if not exists payout_wallet text;

create table if not exists settlements (
  settlement_id uuid primary key default uuid_generate_v4(),
  session_id uuid not null references sessions(session_id) on delete cascade,
  site_id uuid not null references sites(site_id) on delete cascade,
  amount numeric(12,4) not null,
  currency text not null,
  status text not null default 'Pending', -- Pending|Submitted|Confirmed|Failed
  chain text,
  tx_hash text,
  external_ref text,
  error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique(session_id)
);
create index if not exists idx_settlements_status_created on settlements(status, created_at);
