-- Label-based multi-server targeting + per-job resource limits.
--
-- Jobs can now target either a single server (target_kind = 'server', server_id set)
-- or a label selector (target_kind = 'labels', target_labels jsonb). When labels are
-- used, the control plane resolves the matching server set at sync + run-now time.

begin;

-- Allow server_id to be NULL for label-targeted jobs, and change the FK so deleting
-- a server no longer deletes label-targeted jobs that happened to mention it.
alter table jobs alter column server_id drop not null;
alter table jobs drop constraint if exists jobs_server_id_fkey;
alter table jobs
  add constraint jobs_server_id_fkey
  foreign key (server_id) references servers(id) on delete set null;

alter table jobs
  add column if not exists target_kind text not null default 'server';
alter table jobs
  add column if not exists target_labels jsonb not null default '{}'::jsonb;

-- Resource limits. 0 means unlimited. Enforced by the agent via systemd-run when
-- available; otherwise the agent logs and runs unbounded.
alter table jobs
  add column if not exists cpu_quota_percent integer not null default 0;
alter table jobs
  add column if not exists memory_max_mb     integer not null default 0;

-- Helpful index for label-match lookups.
create index if not exists jobs_target_labels_idx on jobs using gin (target_labels);

commit;
