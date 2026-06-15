-- CronCompose initial schema.
-- See docs/data-model.md for the design rationale.

begin;

create table if not exists users (
  id            text primary key,
  email         text unique not null,
  name          text not null,
  password_hash text,
  role          text not null default 'admin',
  created_at    timestamptz not null default now()
);

create table if not exists servers (
  id               text primary key,
  name             text not null,
  description      text,
  os               text,
  arch             text,
  labels           jsonb not null default '{}',
  status           text not null default 'pending',
  agent_version    text,
  cert_fingerprint text,
  last_seen_at     timestamptz,
  created_at       timestamptz not null default now()
);

create table if not exists enrollment_tokens (
  id          text primary key,
  token_hash  text not null,
  server_id   text references servers(id) on delete cascade,
  expires_at  timestamptz not null,
  used_at     timestamptz,
  created_by  text references users(id),
  created_at  timestamptz not null default now()
);

create table if not exists jobs (
  id                 text primary key,
  server_id          text not null references servers(id) on delete cascade,
  name               text not null,
  description        text,
  interpreter        text not null default 'bash',
  schedule_cron      text not null,
  timezone           text not null default 'UTC',
  enabled            boolean not null default true,
  timeout_seconds    integer not null default 3600,
  concurrency_policy text not null default 'skip',
  catchup_policy     text not null default 'once',
  max_retries        integer not null default 0,
  working_dir        text,
  run_as_user        text,
  current_version_id text,
  created_at         timestamptz not null default now(),
  updated_at         timestamptz not null default now()
);

create table if not exists job_versions (
  id             text primary key,
  job_id         text not null references jobs(id) on delete cascade,
  version_number integer not null,
  script_body    text not null,
  env            jsonb not null default '{}',
  secret_refs    jsonb not null default '[]',
  created_by     text references users(id),
  created_at     timestamptz not null default now(),
  unique (job_id, version_number)
);

create table if not exists secrets (
  id         text primary key,
  scope      text not null default 'global',
  scope_id   text,
  name       text not null,
  value_enc  bytea not null,
  created_by text references users(id),
  created_at timestamptz not null default now(),
  unique (scope, scope_id, name)
);

create table if not exists runs (
  id             text primary key,
  job_id         text not null references jobs(id) on delete cascade,
  job_version_id text not null references job_versions(id),
  server_id      text not null references servers(id),
  trigger        text not null,
  status         text not null default 'pending',
  scheduled_for  timestamptz,
  started_at     timestamptz,
  finished_at    timestamptz,
  exit_code      integer,
  duration_ms    integer,
  error          text,
  created_at     timestamptz not null default now()
);

create index if not exists runs_job_created_idx on runs (job_id, created_at desc);
create index if not exists runs_server_status_idx on runs (server_id, status);

create table if not exists run_logs (
  id     bigserial primary key,
  run_id text not null references runs(id) on delete cascade,
  stream text not null,
  seq    integer not null,
  chunk  text not null,
  ts     timestamptz not null default now(),
  unique (run_id, stream, seq)
);

create table if not exists api_tokens (
  id           text primary key,
  user_id      text not null references users(id) on delete cascade,
  name         text not null,
  token_hash   text not null,
  scopes       jsonb not null default '[]',
  last_used_at timestamptz,
  created_at   timestamptz not null default now()
);

create table if not exists audit_log (
  id            bigserial primary key,
  actor_user_id text references users(id),
  action        text not null,
  target_type   text,
  target_id     text,
  metadata      jsonb not null default '{}',
  ts            timestamptz not null default now()
);

commit;
