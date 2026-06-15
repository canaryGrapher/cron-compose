#!/usr/bin/env bash
# CronCompose agent installer. Downloads the binary, drops a systemd unit, enrolls.
#
# Required env (or flags):
#   TOKEN                 one-time enrollment token from the UI (required)
#   CONTROL_PLANE_HTTP    base URL for the REST enroll call, e.g. https://cc.example.com/api/v1
#   CONTROL_PLANE_ADDR    host:port of the mTLS gRPC endpoint, e.g. cc.example.com:9090
#   CONTROL_PLANE_SNI     server name to verify against (defaults to host portion of ADDR)
#   AGENT_VERSION         release tag to download, defaults to "latest"
#   DOWNLOAD_BASE         base URL for the binary; default points at GitHub releases
#
# Run example:
#   curl -sSL https://cc.example.com/agent.sh | \
#     sudo TOKEN=abc CONTROL_PLANE_HTTP=https://cc.example.com/api/v1 \
#          CONTROL_PLANE_ADDR=cc.example.com:9090 bash

set -euo pipefail

: "${TOKEN:?TOKEN env var is required}"
: "${CONTROL_PLANE_HTTP:?CONTROL_PLANE_HTTP env var is required}"
: "${CONTROL_PLANE_ADDR:?CONTROL_PLANE_ADDR env var is required}"

AGENT_VERSION="${AGENT_VERSION:-latest}"
DOWNLOAD_BASE="${DOWNLOAD_BASE:-https://github.com/croncompose/croncompose/releases}"
SNI="${CONTROL_PLANE_SNI:-${CONTROL_PLANE_ADDR%%:*}}"
DATA_DIR="${DATA_DIR:-/var/lib/croncompose}"
BIN_PATH=/usr/local/bin/croncompose-agent
UNIT_PATH=/etc/systemd/system/croncompose-agent.service

if [[ "$(id -u)" -ne 0 ]]; then
  echo "this installer must be run as root (use sudo)" >&2
  exit 1
fi

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) target="linux-amd64" ;;
  aarch64|arm64) target="linux-arm64" ;;
  armv7l) target="linux-armv7" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

echo "==> creating service user and data dir"
id -u croncompose >/dev/null 2>&1 || useradd --system --home-dir "$DATA_DIR" --shell /usr/sbin/nologin croncompose
install -d -o croncompose -g croncompose -m 0700 "$DATA_DIR"

echo "==> downloading agent ($AGENT_VERSION / $target)"
url="${DOWNLOAD_BASE}/${AGENT_VERSION}/download/croncompose-agent-${target}"
curl -fSL --retry 3 -o "$BIN_PATH.tmp" "$url"
chmod 0755 "$BIN_PATH.tmp"
mv "$BIN_PATH.tmp" "$BIN_PATH"

echo "==> writing systemd unit"
cat >"$UNIT_PATH" <<EOF
[Unit]
Description=CronCompose agent
After=network-online.target
Wants=network-online.target

[Service]
User=croncompose
Group=croncompose
Environment=CONTROL_PLANE_ADDR=${CONTROL_PLANE_ADDR}
Environment=CONTROL_PLANE_HTTP=${CONTROL_PLANE_HTTP}
Environment=CONTROL_PLANE_SNI=${SNI}
Environment=DATA_DIR=${DATA_DIR}
ExecStart=${BIN_PATH} run
Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

echo "==> enrolling"
sudo -u croncompose \
  CONTROL_PLANE_ADDR="$CONTROL_PLANE_ADDR" \
  CONTROL_PLANE_HTTP="$CONTROL_PLANE_HTTP" \
  CONTROL_PLANE_SNI="$SNI" \
  DATA_DIR="$DATA_DIR" \
  "$BIN_PATH" enroll --token="$TOKEN"

echo "==> starting service"
systemctl daemon-reload
systemctl enable --now croncompose-agent.service

echo
echo "done. follow logs with: journalctl -u croncompose-agent -f"
