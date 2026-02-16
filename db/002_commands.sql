-- Migration: add commands table (CPMS -> Gateway control plane)
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
