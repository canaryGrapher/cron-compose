# REST API

The Fiber v3 control plane exposes a REST API under `/api/v1` for the Next.js UI and for
external callers. Agents do **not** use this API; they use the gRPC channel in
[agent-protocol.md](agent-protocol.md).

- Auth: browser uses a session cookie; external callers use a bearer `api_token`.
- Content type: JSON.
- Live data (run logs, status) uses SSE endpoints noted below.
- All mutating endpoints write an `audit_log` entry.

## Auth

| Method | Path                | Purpose                              |
|--------|---------------------|--------------------------------------|
| POST   | `/auth/login`       | Email + password, sets session.      |
| POST   | `/auth/logout`      | Clears session.                      |
| GET    | `/me`               | Current user and role.               |

## Servers

| Method | Path                              | Purpose                                  |
|--------|-----------------------------------|------------------------------------------|
| GET    | `/servers`                        | List servers with status and last seen.  |
| POST   | `/servers`                        | Create a server, returns enrollment token + install command. |
| GET    | `/servers/:id`                    | Server detail.                           |
| PATCH  | `/servers/:id`                    | Rename, edit description/labels.         |
| DELETE | `/servers/:id`                    | Remove server (and its jobs).            |
| POST   | `/servers/:id/enrollment-token`   | Re-issue a one-time enrollment token.    |
| POST   | `/servers/:id/revoke`             | Revoke the agent cert; forces re-enroll. |

## Jobs

| Method | Path                  | Purpose                                          |
|--------|-----------------------|--------------------------------------------------|
| GET    | `/jobs?server=:id`    | List jobs, optionally filtered by server.        |
| POST   | `/jobs`               | Create a job (creates version 1). Body: server_id, name, interpreter, script_body, schedule_cron, timezone, options. |
| GET    | `/jobs/:id`           | Job detail incl. current version.                |
| PATCH  | `/jobs/:id`           | Edit metadata/schedule/options. Editing `script_body` creates a new `job_version`. |
| DELETE | `/jobs/:id`           | Delete a job.                                    |
| POST   | `/jobs/:id/enable`    | Enable scheduling.                              |
| POST   | `/jobs/:id/disable`   | Disable scheduling.                            |
| POST   | `/jobs/:id/run`       | Trigger a manual run now (sends `RunNow` to agent). |

## Job versions

| Method | Path                          | Purpose                            |
|--------|-------------------------------|------------------------------------|
| GET    | `/jobs/:id/versions`          | List version history.              |
| GET    | `/jobs/:id/versions/:n`       | Get a specific version's script.   |
| POST   | `/jobs/:id/versions/:n/restore` | Make an old version current (creates a new version from it). |

## Runs and logs

| Method | Path                          | Purpose                                  |
|--------|-------------------------------|------------------------------------------|
| GET    | `/jobs/:id/runs`              | Run history for a job (paginated).       |
| GET    | `/runs/:id`                   | Run detail: status, timing, exit code.   |
| GET    | `/runs/:id/logs`             | Full captured log (text), for finished runs. |
| GET    | `/runs/:id/logs/stream`      | **SSE** live log stream for an in-progress run. |
| POST   | `/runs/:id/cancel`           | Cancel a running job (sends `CancelRun`). |

## Secrets

| Method | Path             | Purpose                                          |
|--------|------------------|--------------------------------------------------|
| GET    | `/secrets`       | List secret names and scopes (never values).     |
| POST   | `/secrets`       | Create a secret (name, scope, value). Value is encrypted at rest. |
| DELETE | `/secrets/:id`   | Delete a secret.                                 |

## API tokens

| Method | Path                 | Purpose                                  |
|--------|----------------------|------------------------------------------|
| GET    | `/api-tokens`        | List the caller's tokens (no secrets).   |
| POST   | `/api-tokens`        | Create a token; plaintext shown once.    |
| DELETE | `/api-tokens/:id`    | Revoke a token.                          |

## Audit

| Method | Path        | Purpose                              |
|--------|-------------|--------------------------------------|
| GET    | `/audit`    | Paginated, filterable audit entries. |

## Conventions

- Pagination: `?limit=&cursor=` with an opaque cursor; list responses return
  `{ items: [...], next_cursor }`.
- Errors: JSON `{ error: { code, message } }` with appropriate HTTP status.
- Timestamps: RFC 3339 / ISO 8601 in UTC.
- Idempotency: run upserts from agents are keyed by run `id`; the manual-run endpoint
  may accept an `Idempotency-Key` header to avoid duplicate triggers.
