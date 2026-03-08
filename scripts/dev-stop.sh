#!/bin/bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common-dev.sh"

if [ -f "$PIDS_FILE" ]; then
  load_env_file "$PIDS_FILE"
fi

if [ -f "$PORTS_FILE" ]; then
  load_env_file "$PORTS_FILE"
fi

stopped=0

if process_running "${WEB_PID:-}"; then
  log "stopping web server ($WEB_PID)"
  stop_process_tree "$WEB_PID"
  stopped=1
elif [ -n "${WEB_PORT:-}" ]; then
  for pid in $(lsof -nP -iTCP:"$WEB_PORT" -sTCP:LISTEN -t 2>/dev/null || true); do
    log "stopping web server by port $WEB_PORT ($pid)"
    stop_process_tree "$pid"
    stopped=1
  done
fi

if process_running "${SERVER_PID:-}"; then
  log "stopping api server ($SERVER_PID)"
  stop_process_tree "$SERVER_PID"
  stopped=1
elif [ -n "${SERVER_PORT:-}" ]; then
  for pid in $(lsof -nP -iTCP:"$SERVER_PORT" -sTCP:LISTEN -t 2>/dev/null || true); do
    log "stopping api server by port $SERVER_PORT ($pid)"
    stop_process_tree "$pid"
    stopped=1
  done
fi

clear_pid_file

if [ "$stopped" -eq 1 ]; then
  log "dev stack stopped"
else
  log "dev stack was not running"
fi
