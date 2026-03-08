#!/bin/bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common-dev.sh"

[ -f "$PORTS_FILE" ] || die "no dev stack state found"

load_env_file "$PORTS_FILE"
if [ -f "$PIDS_FILE" ]; then
  load_env_file "$PIDS_FILE"
fi

if process_running "${SERVER_PID:-}" || process_running "${WEB_PID:-}"; then
  log "dev stack is running"
else
  log "dev stack is not running"
fi

show_runtime_summary

if [ -n "${SERVER_PID:-}" ]; then
  log "server pid: $SERVER_PID"
fi
if [ -n "${WEB_PID:-}" ]; then
  log "web pid: $WEB_PID"
fi
