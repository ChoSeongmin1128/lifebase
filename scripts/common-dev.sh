#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
STATE_DIR="$PROJECT_DIR/tmp/dev-stack"
LOG_DIR="$STATE_DIR/logs"
PORTS_FILE="$STATE_DIR/ports.env"
PIDS_FILE="$STATE_DIR/pids.env"
SERVER_LOG="$LOG_DIR/server.log"
WEB_LOG="$LOG_DIR/web.log"

mkdir -p "$LOG_DIR"

log() {
  printf '%s\n' "$*"
}

warn() {
  printf 'warning: %s\n' "$*" >&2
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

load_env_file() {
  local file="$1"
  if [ -f "$file" ]; then
    # shellcheck disable=SC1090
    . "$file"
  fi
}

process_running() {
  local pid="${1:-}"
  [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null
}

port_in_use() {
  local port="$1"
  lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
}

find_free_port() {
  local port="$1"
  while port_in_use "$port"; do
    port=$((port + 1))
  done
  printf '%s\n' "$port"
}

assign_ports() {
  local preferred_server preferred_web

  preferred_server="${LIFEBASE_SERVER_PORT:-38117}"
  preferred_web="${LIFEBASE_WEB_PORT:-39001}"

  SERVER_PORT="$(find_free_port "$preferred_server")"
  WEB_PORT="$(find_free_port "$preferred_web")"

  API_URL="http://localhost:$SERVER_PORT"
  WEB_URL="http://localhost:$WEB_PORT"
  ADMIN_URL="$WEB_URL"
}

write_ports_file() {
  cat >"$PORTS_FILE" <<EOF
SERVER_PORT=$SERVER_PORT
WEB_PORT=$WEB_PORT
API_URL=$API_URL
WEB_URL=$WEB_URL
ADMIN_URL=$ADMIN_URL
EOF
}

write_pids_file() {
  cat >"$PIDS_FILE" <<EOF
SERVER_PID=$SERVER_PID
WEB_PID=$WEB_PID
EOF
}

clear_pid_file() {
  rm -f "$PIDS_FILE"
}

worktree_label() {
  basename "$PROJECT_DIR"
}

show_runtime_summary() {
  log "worktree: $(worktree_label)"
  log "web: $WEB_URL"
  log "api: $API_URL"
  log "server log: $SERVER_LOG"
  log "web log: $WEB_LOG"
}

wait_for_port() {
  local port="$1"
  local name="$2"
  local pid="${3:-}"
  local retries=30

  while [ "$retries" -gt 0 ]; do
    if port_in_use "$port"; then
      return 0
    fi
    if [ -n "$pid" ] && ! process_running "$pid"; then
      warn "$name exited before opening port $port"
      return 1
    fi
    sleep 1
    retries=$((retries - 1))
  done

  warn "$name did not open port $port in time"
  return 1
}

collect_descendants() {
  local pid="$1"
  local child

  for child in $(pgrep -P "$pid" 2>/dev/null || true); do
    collect_descendants "$child"
    printf '%s\n' "$child"
  done
}

stop_process_tree() {
  local pid="${1:-}"
  local descendants child

  if ! process_running "$pid"; then
    return 0
  fi

  descendants="$(collect_descendants "$pid")"
  for child in $descendants; do
    kill -TERM "$child" 2>/dev/null || true
  done
  kill -TERM "$pid" 2>/dev/null || true

  sleep 1

  for child in $descendants; do
    if process_running "$child"; then
      kill -KILL "$child" 2>/dev/null || true
    fi
  done
  if process_running "$pid"; then
    kill -KILL "$pid" 2>/dev/null || true
  fi
}

detect_main_worktree() {
  git -C "$PROJECT_DIR" worktree list --porcelain 2>/dev/null | awk '
    $1 == "worktree" {
      print substr($0, 10)
      exit
    }
  '
}
