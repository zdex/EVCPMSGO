-- Migration: sites + tariffs (per kWh) + session pricing fields
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
