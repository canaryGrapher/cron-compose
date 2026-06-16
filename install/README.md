# CronCompose installer

Interactive, from-source installers that stand up the whole CronCompose **control
plane** (PostgreSQL + the Go API + the Next.js web UI) on one machine, and on
Linux/macOS optionally enroll a local **agent** so the box is also a job runner.

- **Linux / macOS:** `install/install.sh`
- **Windows:** `install/install.ps1` (control plane only: see [Agent](#agent))

The installer asks for the ports to use (suggesting a free one for each), collects the
required environment variables, generates the secrets for you, builds everything,
creates the database schema, starts the services, and prints how to sign in.

On Linux/macOS it also builds and runs the in-repo **single-entry reverse proxy**, so the
whole system sits behind one public port: `/app` serves the UI, `/api` the REST API, and
agent gRPC is passed through. The bare URL redirects to `/app`. Pass `--no-proxy` to keep
the old layout where the UI and API are exposed on separate ports. (The Windows installer
does not run the proxy yet; it serves the UI directly under `/app`.)

## Prerequisites

You need a working toolchain because this builds from source:

- **Go 1.25+**: builds the API, the migration tool, the agent, and the `cc` CLI.
- **Node 20+** and **npm**: builds and runs the web UI.
- **PostgreSQL**: optional to install yourself. The installer can **install and configure
  it for you** via your OS package manager (apt, dnf/yum, pacman, apk, zypper, or
  Homebrew), or you can point it at an existing server, a `psql`-reachable local server,
  or Docker.

The installer checks all of this up front and stops with a clear message if something
is missing.

## Quick start

Linux / macOS:

```sh
git clone <repo> && cd croncompose
./install/install.sh
```

Windows (PowerShell):

```powershell
git clone <repo>; cd croncompose
./install/install.ps1
```

Answer the prompts and you'll end up with the UI behind the single proxy port you chose
(open `http://<host>:<proxy port>`; it redirects to `/app`).

## What it asks you

1. **Runtime directory**: where logs, pids, TLS material, and agent data live
   (default `./.run`).
2. **Advertise host**: the hostname or IP that browsers and agents use to reach this
   box (default `localhost`).
3. **Ports**: the public proxy port (8000), and the internal web UI (3000), REST API
   (8080), and agent gRPC (9090). For each, the installer probes for a free port and
   offers it; press Enter to accept or type your own. Occupied ports are flagged before
   you commit. With the proxy on, only the proxy port needs to be reachable from outside;
   the others are bound to loopback (web) or used internally. With `--no-proxy` you are
   not asked for a proxy port.
4. **Admin account**: the email and password you'll sign in with. Leave the password
   blank and one is generated and shown to you.
5. **Database**: pick existing / psql / Docker (see [Database options](#database-options)).
6. **OIDC SSO**: optional; if you opt in it collects the four `OIDC_*` values.
7. **Extra variables**: optionally add any other `KEY=VALUE` pairs to append to `.env`.

`SESSION_SECRET` and `SECRETS_MASTER_KEY` are always generated for you with a CSPRNG.
Everything is written to a `.env` file (mode `600`) at the repo root.

## Database options

- **Install it for me** (offered if a package manager is found; the default): the
  installer installs the PostgreSQL server with your OS package manager, starts and
  enables the service (running `initdb` where the package manager doesn't), then creates
  the role and database. You just choose the database name, role, and password (a
  password is generated if you leave it blank). Needs `sudo` on Linux. Idempotent: it
  skips the install if a server is already present. Supported managers: `apt`, `dnf`,
  `yum`, `pacman`, `apk`, `zypper`, and Homebrew (`brew`). Preview the exact commands
  without running them by setting `CC_DB_DRY_RUN=1`.
- **Existing**: you paste a `DATABASE_URL`. Nothing is created; the schema is applied
  to whatever you point at.
- **psql** (offered if `psql` is on PATH): you give a superuser connection and the
  installer creates the role and database in an already-running server.
- **Docker** (offered if `docker` is on PATH): the installer runs Postgres from
  `docker-compose.yml`, waits for it to be ready, and uses it.

Migrations are applied by a small bundled tool (`control-plane/cmd/migrate`) that talks
to Postgres directly, so **no `psql` client is required** on Windows or macOS. It records
applied files in a `schema_migrations` table, so re-running the installer is safe.

## Managing the stack

The installer generates a control script next to `.env` and uses it to start things, so
there is one command you keep using afterwards:

```sh
./croncompose-ctl.sh status       # what's running (incl. the proxy)
./croncompose-ctl.sh logs proxy   # tail a service log (control-plane | web | proxy | agent)
./croncompose-ctl.sh restart      # restart everything
./croncompose-ctl.sh stop         # stop everything
./croncompose-ctl.sh start        # start (or resume) everything
```

On Windows the equivalent is `./croncompose-ctl.ps1 <action>`.

Services run as background processes. To survive reboots, wrap the control script in a
`systemd` service (Linux), a `launchd` agent (macOS), or a scheduled task (Windows).

## Changing the external address (single point of change)

With the proxy on, the browser, the REST API, and agents all reach the box through the
single proxy port; the proxy forwards to the web UI and API internally over `127.0.0.1`.
Use any hostname that reaches the box (`localhost`, `raspberrypi.local`, your domain).

The address matters where the control plane advertises *itself* (the agent install
command, the gRPC address agents dial, the OIDC redirect, the TLS SAN). The installer
writes two `.env` lines, both pointing at the proxy:

```
PUBLIC_BASE_URL=http://raspberrypi.local:8000    # derives REST URL, OIDC redirect, TLS SAN
PUBLIC_GRPC_ADDR=raspberrypi.local:8000          # agents dial the proxy; it passes gRPC through
```

To move to another address (for example `https://cron.example.com`), change the host/port
in **both** lines and restart (`./croncompose-ctl.sh restart`); add the new host to
`TLS_HOSTS`, or delete `<runtime>/tls` to regenerate the server cert so its SAN covers it.
With `--no-proxy` there is no `PUBLIC_GRPC_ADDR` line: `PUBLIC_BASE_URL` alone drives
everything and the gRPC address derives from it plus the internal gRPC port.

## Agent

The per-server agent runs scheduled jobs through a Unix shell and uses Unix process
APIs, so it builds and runs on **Linux and macOS only**. The Linux/macOS installer can
enroll and start a local agent for you (it logs in with the seeded admin, mints a
one-time token, enrolls, and starts the agent). On Windows the installer sets up the
control plane only; run agents on Linux/macOS hosts and point them at this control
plane (see `scripts/install-agent.sh`).

## Non-interactive install

Pass `--non-interactive` (PowerShell: `-NonInteractive`) to take defaults without
prompting. Values come from `CC_*` environment variables:

| Variable             | Meaning                                            | Default                          |
|----------------------|----------------------------------------------------|----------------------------------|
| `CC_RUNTIME_DIR`     | runtime state directory                            | `./.run`                         |
| `CC_ADVERTISE_HOST`  | host used to build the default `PUBLIC_BASE_URL`   | `localhost`                      |
| `CC_PUBLIC_BASE_URL` | external base URL written to `.env` (overrides the host-derived default) | `http://<advertise>:<proxy port>` (proxy on), else `:<api port>` |
| `CC_PROXY_PORT`      | public single-entry proxy port                     | first free at/after `8000`       |
| `CC_WEB_PORT`        | web UI port (internal when proxied)                | first free at/after `3000`       |
| `CC_API_PORT`        | REST API port                                      | first free at/after `8080`       |
| `CC_GRPC_PORT`       | agent gRPC port                                    | first free at/after `9090`       |
| `CC_ADMIN_EMAIL`     | seed admin email                                   | `admin@example.com`              |
| `CC_ADMIN_PASSWORD`  | seed admin password                                | generated if empty               |
| `CC_DB_METHOD`       | `native` \| `existing` \| `psql` \| `docker`        | `native` if a package manager is found, else `existing` |
| `CC_DATABASE_URL`    | DSN (for `existing`)                               | local dev DSN                    |
| `CC_DB_NAME` / `CC_DB_USER` / `CC_DB_PASS` | database, role, password (for `native`/`psql`) | `croncompose` / `croncompose` / generated |
| `CC_LOG_LEVEL`       | `debug` \| `info` \| `warn` \| `error`             | `info`                           |

Other flags: `--no-proxy` (skip the single-entry proxy; expose web/API on separate ports),
`--no-web` / `-NoWeb` (API-only), `--no-agent` (Linux/macOS, control plane only),
`--runtime-dir DIR` / `-RuntimeDir DIR`.

Example: a headless, scripted install against an existing database:

```sh
CC_DB_METHOD=existing \
CC_DATABASE_URL='postgres://cc:cc@db:5432/cc?sslmode=disable' \
CC_ADMIN_EMAIL=you@example.com CC_ADMIN_PASSWORD='strong-pass' \
CC_PROXY_PORT=8000 CC_WEB_PORT=3000 CC_API_PORT=8080 CC_GRPC_PORT=9090 \
./install/install.sh --non-interactive
```

## Generated files

- `.env`: config and secrets (mode `600`, git-ignored). Source of truth for the stack.
- `croncompose-ctl.sh` / `croncompose-ctl.ps1`: process manager (git-ignored).
- `.run/`: logs, pids, TLS material, and agent data (git-ignored).
- `control-plane/bin/`, `cli/bin/`, `agent/bin/`, `proxy/bin/`: compiled binaries (git-ignored).

## Production notes

This is a from-source install meant to get you running quickly. For real deployments:
set **Advertise host** to a real DNS name, terminate TLS in front of the API and UI,
replace the self-signed CA under `<runtime>/tls` with your own PKI, and run the
services under a process supervisor. The repository's `docker-compose.prod.yml` is a
container-based alternative.

## Troubleshooting

- **"Go/Node is required"**: install the toolchain and re-run; the installer prints the
  download links.
- **Web didn't bind its port**: check `./croncompose-ctl.sh logs web`. The UI is built
  with `output: "standalone"` and runs as `node .next/standalone/server.js`.
- **Migrations failed**: verify the `DATABASE_URL` is reachable and the role can create
  tables. The migration tool retries the connection briefly on startup.
- **Blank page or 404 at the root**: open `/app`, not `/`. The proxy redirects the bare
  URL to `/app`; if you used `--no-proxy`, browse `http://<host>:<web port>/app` directly.
- **Proxy won't start**: check `./croncompose-ctl.sh logs proxy`. It needs the web and
  control-plane ports free for its upstreams; the port it listens on is `CC_PROXY_PORT`.
