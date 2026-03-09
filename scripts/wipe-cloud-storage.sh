#!/bin/bash
set -euo pipefail

if [ "${1:-}" != "--yes" ]; then
  echo "usage: $0 --yes" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

export SERVER_ENV="${SERVER_ENV:-production}"

"$SCRIPT_DIR/check-recent-db-backup.sh"

DATA_PATH="${STORAGE_DATA_PATH:-/Volumes/WDRedPlus/LifeBase/data}"
THUMB_PATH="${STORAGE_THUMB_PATH:-/Users/seongmin/lifebase-cache/thumbs}"

find "$DATA_PATH" -mindepth 1 -delete
find "$THUMB_PATH" -mindepth 1 -delete

echo "cloud_storage_wiped data_path=$DATA_PATH thumb_path=$THUMB_PATH"
