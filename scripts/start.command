#!/bin/bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_FILE="$PROJECT_DIR/scripts/lifebase.log"

log() {
    local msg="$(date '+%Y-%m-%d %H:%M:%S'): $1"
    echo "$msg"
    echo "$msg" >> "$LOG_FILE"
}

log "===== LifeBase 서비스 시작 ====="

# 1. Go API 서버 (포트 38117)
if lsof -nP -iTCP:38117 -sTCP:LISTEN &>/dev/null; then
    log "API 서버가 이미 실행 중 (포트 38117)"
else
    log "API 서버 시작 중..."
    cd "$PROJECT_DIR/apps/server"
    nohup go run ./cmd/server/ >> "$LOG_FILE" 2>&1 &
    log "API 서버 시작됨 (PID: $!)"
fi

# 2. Next.js 웹 서버 (포트 39001)
if lsof -nP -iTCP:39001 -sTCP:LISTEN &>/dev/null; then
    log "웹 서버가 이미 실행 중 (포트 39001)"
else
    log "웹 서버 시작 중..."
    cd "$PROJECT_DIR/apps/web"
    nohup npm run start >> "$LOG_FILE" 2>&1 &
    log "웹 서버 시작됨 (PID: $!)"
fi

log "===== LifeBase 서비스 시작 완료 ====="
