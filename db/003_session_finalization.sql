-- Migration: session finalization fields for billing/tokenization readiness
alter table sessions
  add column if not exists energy_wh bigint,
  add column if not exists energy_source text,
  add column if not exists is_estimated boolean not null default false,
  add column if not exists finalized_at timestamptz;

-- Optional index to find finalized sessions quickly
create index if not exists idx_sessions_finalized_at on sessions(finalized_at);
