#!/bin/bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_FILE="$PROJECT_DIR/scripts/lifebase.log"

log() {
    local msg="$(date '+%Y-%m-%d %H:%M:%S'): $1"
    echo "$msg"
    echo "$msg" >> "$LOG_FILE"
}

log "===== LifeBase 서비스 종료 ====="

# 1. Next.js 웹 서버 (포트 39001)
WEB_PID=$(lsof -nP -iTCP:39001 -sTCP:LISTEN -t 2>/dev/null || true)
if [ -n "$WEB_PID" ]; then
    log "웹 서버 종료 중 (PID: $WEB_PID)..."
    kill "$WEB_PID" 2>/dev/null || true
    log "웹 서버 종료 완료"
else
    log "웹 서버 미실행 - 건너뜀"
fi

# 2. Go API 서버 (포트 38117)
API_PID=$(lsof -nP -iTCP:38117 -sTCP:LISTEN -t 2>/dev/null || true)
if [ -n "$API_PID" ]; then
    log "API 서버 종료 중 (PID: $API_PID)..."
    kill "$API_PID" 2>/dev/null || true
    log "API 서버 종료 완료"
else
    log "API 서버 미실행 - 건너뜀"
fi

log "===== LifeBase 서비스 종료 완료 ====="
