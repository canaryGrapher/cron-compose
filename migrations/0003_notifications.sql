-- Webhook notification targets. Fires on any non-success Run status.

begin;

create table if not exists notification_targets (
  id           text primary key,
  name         text not null,
  kind         text not null default 'webhook',  -- room to grow (slack, email)
  url          text not null,
  enabled      boolean not null default true,
  created_at   timestamptz not null default now()
);

commit;
