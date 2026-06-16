# croncompose-proxy

A tiny, dependency-free Go reverse proxy that gives CronCompose a **single entry
point**. Point one URL at this proxy and it fans traffic out internally:

| Inbound                          | Routed to                                  |
| -------------------------------- | ------------------------------------------ |
| `/app/*` (the UI)                | web UI (Next.js, `basePath: /app`)         |
| `/api/*`                         | control-plane REST (`/api` → `/api/v1`)    |
| `/` and anything else            | `302` into `/app` (so the bare domain opens the app) |
| agent gRPC (mTLS)                | control-plane gRPC, passed through untouched |

The UI prefix (`/app`) must match the Next.js `basePath`. Set `WEB_PREFIX=/`
to serve the UI at the root instead (and drop `basePath` from `next.config.ts`).

Everything shares one listener. The proxy looks at the first bytes of each
connection and decides:

- **Cleartext HTTP** (`GET …`) → HTTP reverse proxy (UI + REST).
- **TLS handshake** (`0x16`) → check SNI:
  - SNI = `AGENT_SNI` (or no browser cert configured) → **raw passthrough** to
    the control-plane gRPC port. TLS is never decrypted, so agent **mTLS stays
    end-to-end**.
  - otherwise → terminate browser HTTPS with the proxy's cert, then HTTP route.

## Configuration (env)

| Var                  | Default                  | Meaning                                              |
| -------------------- | ------------------------ | ---------------------------------------------------- |
| `PROXY_LISTEN_ADDR`  | `:8000`                  | the single public listener (`:443` in TLS mode)      |
| `WEB_UPSTREAM`       | `http://localhost:3000`  | Next.js UI                                           |
| `API_UPSTREAM`       | `http://localhost:8080`  | control-plane REST                                   |
| `GRPC_UPSTREAM`      | `localhost:9090`         | control-plane gRPC (agents)                          |
| `API_PREFIX`         | `/api`                   | public REST prefix                                   |
| `API_UPSTREAM_PREFIX`| `/api/v1`                | control-plane REST prefix it rewrites to             |
| `WEB_PREFIX`         | `/app`                   | UI path prefix (match Next `basePath`); `/` = root   |
| `TLS_CERT_FILE`      | _(unset)_                | PEM cert; enables browser-HTTPS termination          |
| `TLS_KEY_FILE`       | _(unset)_                | PEM key (set together with the cert)                 |
| `AGENT_SNI`          | _(unset)_                | SNI agents present; required when TLS is enabled     |

## Two ways to run

**Plain (one port, no cert at the proxy).** Browsers use `http://host:8000`,
agents dial `host:8000` with mTLS. The proxy tells them apart by the first byte
(ASCII request vs. TLS record), so a single port carries the UI, the API, and
agent gRPC. Terminate browser TLS upstream (or run on a trusted network).

```
PROXY_LISTEN_ADDR=:8000 \
WEB_UPSTREAM=http://web:3000 \
API_UPSTREAM=http://control-plane:8080 \
GRPC_UPSTREAM=control-plane:9090 \
./proxy
```

**TLS termination (one port, browser HTTPS at the proxy).** Browsers hit
`https://app.example.com`; agents hit `https://agents.example.com` — same IP,
same `:443`, separated by SNI.

```
PROXY_LISTEN_ADDR=:443 \
TLS_CERT_FILE=/tls/fullchain.pem TLS_KEY_FILE=/tls/key.pem \
AGENT_SNI=agents.example.com \
WEB_UPSTREAM=http://web:3000 \
API_UPSTREAM=http://control-plane:8080 \
GRPC_UPSTREAM=control-plane:9090 \
./proxy
```

## Pointing agents at the proxy

The control plane advertises a gRPC address to agents at enrollment. Set it to
the proxy's public address so agents connect through the single entry point:

```
PUBLIC_GRPC_ADDR=app.example.com:443     # or host:8000 in plain mode
PUBLIC_HTTP_URL=https://app.example.com/api/v1
```

## Health

`GET /__health` returns `200 ok` from the proxy itself (used by the container
health check).

## Build & test

```
cd proxy
go build ./...
go vet ./...
```

No third-party dependencies — standard library only.
