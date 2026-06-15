#!/bin/sh
# Stop and disable the service before the package files are removed.
set -e

if command -v systemctl >/dev/null 2>&1; then
  systemctl stop croncompose-agent.service || true
  systemctl disable croncompose-agent.service || true
fi
