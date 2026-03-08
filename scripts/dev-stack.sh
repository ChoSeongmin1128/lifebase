#!/bin/bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common-dev.sh"

if [ ! -d "$PROJECT_DIR/node_modules" ]; then
  log "node_modules missing, running pnpm install"
  (cd "$PROJECT_DIR" && pnpm install)
fi

if lsof "$PROJECT_DIR/apps/web/.next/dev/lock" >/dev/null 2>&1; then
  die "next dev is already running for this worktree; stop the existing web process first"
fi

if [ -f "$PIDS_FILE" ]; then
  load_env_file "$PIDS_FILE"
  if process_running "${SERVER_PID:-}" || process_running "${WEB_PID:-}"; then
    load_env_file "$PORTS_FILE"
    log "dev stack already running"
    show_runtime_summary
    exit 0
  fi
  clear_pid_file
fi

assign_ports
write_ports_file

if [ "$SERVER_PORT" != "${LIFEBASE_SERVER_PORT:-38117}" ] || [ "$WEB_PORT" != "${LIFEBASE_WEB_PORT:-39001}" ]; then
  warn "default ports are busy, using api=$SERVER_PORT web=$WEB_PORT"
  warn "Google OAuth local redirect URIs may need to match these ports"
fi

: >"$SERVER_LOG"
: >"$WEB_LOG"

log "starting api server"
(
  cd "$PROJECT_DIR/apps/server" || exit 1
  export SERVER_PORT WEB_URL ADMIN_URL API_URL
  nohup go run ./cmd/server/ >>"$SERVER_LOG" 2>&1 &
  echo $! >"$STATE_DIR/.server.pid"
)
SERVER_PID="$(cat "$STATE_DIR/.server.pid")"
rm -f "$STATE_DIR/.server.pid"

log "starting web server"
(
  cd "$PROJECT_DIR/apps/web" || exit 1
  export NEXT_PUBLIC_API_URL="$API_URL"
  export WEB_URL ADMIN_URL API_URL
  nohup "$PROJECT_DIR/apps/web/node_modules/.bin/next" dev -p "$WEB_PORT" >>"$WEB_LOG" 2>&1 &
  echo $! >"$STATE_DIR/.web.pid"
)
WEB_PID="$(cat "$STATE_DIR/.web.pid")"
rm -f "$STATE_DIR/.web.pid"

write_pids_file

if ! wait_for_port "$SERVER_PORT" "api server" "$SERVER_PID"; then
  warn "last api log lines:"
  tail -n 20 "$SERVER_LOG" || true
  "$SCRIPT_DIR/dev-stop.sh" >/dev/null 2>&1 || true
  exit 1
fi

if ! wait_for_port "$WEB_PORT" "web server" "$WEB_PID"; then
  warn "last web log lines:"
  tail -n 20 "$WEB_LOG" || true
  "$SCRIPT_DIR/dev-stop.sh" >/dev/null 2>&1 || true
  exit 1
fi

log "dev stack is ready"
show_runtime_summary
