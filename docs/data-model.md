# Data model

PostgreSQL. IDs are ULIDs stored as `text` (sortable, agent-generatable) unless noted.
Timestamps are `timestamptz`. JSON fields are `jsonb`.

## Entities at a glance

- **user** — a person who logs into the control plane.
- **server** — a target machine running one agent. 1:1 with an agent in the MVP.
- **enrollment_token** — a short-lived secret used once to enroll an agent onto a server.
- **job** — a scheduled unit: which server, interpreter, schedule, run options. Points
  at its current `job_version`.
- **job_version** — an immutable snapshot of a job's script and non-secret env. Editing
  a job's script creates a new version, giving you history and rollback.
- **secret** — an encrypted value (API key, password) referenced by jobs at run time.
- **run** — one execution of a job: trigger, status, timing, exit code.
- **run_log** — captured output chunks for a run (stdout/stderr).
- **api_token** — a token for programmatic access to the REST API.
- **audit_log** — who did what, for security and debugging.

Multi-tenancy (organizations/teams) is deferred. The MVP assumes a single tenant; an
`org_id` column can be added later without reshaping these tables.

## Schema (MVP)

```sql
-- People
create table users (
  id            text primary key,
  email         text unique not null,
  name          text not null,
  password_hash text,                       -- null if SSO-only later
  role          text not null default 'admin', -- see security.md RBAC
  created_at    timestamptz not null default now()
);

-- A target machine (one agent each, for now)
create table servers (
  id              text primary key,
  name            text not null,
  description     text,
  os              text,                      -- reported by agent
  arch            text,                      -- reported by agent
  labels          jsonb not null default '{}',
  status          text not null default 'pending', -- pending|online|offline
  agent_version   text,
  cert_fingerprint text,                     -- bound at enrollment, identifies the agent
  last_seen_at    timestamptz,
  created_at      timestamptz not null default now()
);

-- One-time enrollment tokens
create table enrollment_tokens (
  id          text primary key,
  token_hash  text not null,                 -- store only the hash
  server_id   text references servers(id) on delete cascade,
  expires_at  timestamptz not null,
  used_at     timestamptz,
  created_by  text references users(id),
  created_at  timestamptz not null default now()
);

-- A scheduled job, bound to one server in the MVP
create table jobs (
  id                 text primary key,
  server_id          text not null references servers(id) on delete cascade,
  name               text not null,
  description        text,
  interpreter        text not null default 'bash',   -- bash|sh|python3|node
  schedule_cron      text not null,                  -- 5- or 6-field cron
  timezone           text not null default 'UTC',    -- IANA tz, evaluated by agent
  enabled            boolean not null default true,
  timeout_seconds    integer not null default 3600,
  concurrency_policy text not null default 'skip',   -- skip|allow|queue
  catchup_policy     text not null default 'once',   -- once|all|skip (after offline)
  max_retries        integer not null default 0,
  working_dir        text,
  run_as_user        text,
  current_version_id text,                            -- -> job_versions.id
  created_at         timestamptz not null default now(),
  updated_at         timestamptz not null default now()
);

-- Immutable script snapshots (history + rollback)
create table job_versions (
  id            text primary key,
  job_id        text not null references jobs(id) on delete cascade,
  version_number integer not null,
  script_body   text not null,
  env           jsonb not null default '{}',   -- non-secret env vars
  secret_refs   jsonb not null default '[]',   -- names of secrets to inject
  created_by    text references users(id),
  created_at    timestamptz not null default now(),
  unique (job_id, version_number)
);

-- Encrypted secrets, injected as env at run time
create table secrets (
  id            text primary key,
  scope         text not null default 'global', -- global|server|job
  scope_id      text,                           -- server_id or job_id when scoped
  name          text not null,                  -- env var name, e.g. API_KEY
  value_enc     bytea not null,                 -- AES-GCM ciphertext (see security.md)
  created_by    text references users(id),
  created_at    timestamptz not null default now(),
  unique (scope, scope_id, name)
);

-- One execution of a job
create table runs (
  id              text primary key,            -- ULID, may be agent-generated
  job_id          text not null references jobs(id) on delete cascade,
  job_version_id  text not null references job_versions(id),
  server_id       text not null references servers(id),
  trigger         text not null,               -- schedule|manual|api
  status          text not null default 'pending',
                  -- pending|running|succeeded|failed|timed_out|canceled|skipped
  scheduled_for   timestamptz,
  started_at      timestamptz,
  finished_at     timestamptz,
  exit_code       integer,
  duration_ms     integer,
  error           text,
  created_at      timestamptz not null default now()
);

create index on runs (job_id, created_at desc);
create index on runs (server_id, status);

-- Captured output, chunked for streaming, capped per run
create table run_logs (
  id        bigserial primary key,
  run_id    text not null references runs(id) on delete cascade,
  stream    text not null,                     -- stdout|stderr
  seq       integer not null,                  -- ordering within the run
  chunk     text not null,
  ts        timestamptz not null default now(),
  unique (run_id, stream, seq)
);

-- Programmatic API access
create table api_tokens (
  id           text primary key,
  user_id      text not null references users(id) on delete cascade,
  name         text not null,
  token_hash   text not null,
  scopes       jsonb not null default '[]',
  last_used_at timestamptz,
  created_at   timestamptz not null default now()
);

-- Audit trail
create table audit_log (
  id          bigserial primary key,
  actor_user_id text references users(id),
  action      text not null,                   -- e.g. job.create, run.cancel
  target_type text,                            -- server|job|run|secret
  target_id   text,
  metadata    jsonb not null default '{}',
  ts          timestamptz not null default now()
);
```

## Notes on key choices

- **Job versions are immutable.** Editing a script never mutates a row; it inserts a new
  `job_versions` row and bumps `jobs.current_version_id`. A `run` always references the
  exact `job_version_id` it executed, so history stays accurate even after edits.
- **Run IDs can be agent-generated** (ULID). Because the agent runs offline and uploads
  later, the control plane upserts runs by `id` (idempotent) rather than assuming it
  created them.
- **Log storage is capped in Postgres for the MVP.** `run_logs` holds chunked output up
  to a per-run byte limit, then truncates with a marker. Large/long-running output moving
  to object storage or files is a later phase (see roadmap).
- **Secrets store ciphertext only** (`value_enc`), never plaintext. Decryption happens in
  the control plane just before the definition is synced to the agent over the encrypted
  stream. See [security.md](security.md).
