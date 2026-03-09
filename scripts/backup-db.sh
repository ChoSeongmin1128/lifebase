#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

export SERVER_ENV="${SERVER_ENV:-production}"
export DB_BACKUP_ROOT="${DB_BACKUP_ROOT:-/Volumes/WDRedPlus/LifeBase/backups}"

cd "$PROJECT_DIR/apps/server"
go run ./cmd/db-backup-auto/
