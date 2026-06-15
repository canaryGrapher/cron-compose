-- Phase 1 MVP simplification: agents authenticate to AgentStream with a bearer token
-- instead of mTLS. The plaintext token is shown once at Enroll; only the hash is stored.
-- Production swaps to mTLS using the cert_fingerprint column (see docs/security.md).

begin;

alter table servers
  add column if not exists agent_token_hash text;

commit;
