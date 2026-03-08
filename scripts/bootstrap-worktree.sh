#!/bin/bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common-dev.sh"

FORCE_COPY=0
RUN_INSTALL=1
SOURCE_REPO=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --force)
      FORCE_COPY=1
      ;;
    --skip-install)
      RUN_INSTALL=0
      ;;
    --source)
      shift
      [ "$#" -gt 0 ] || die "--source requires a path"
      SOURCE_REPO="$1"
      ;;
    *)
      die "unknown option: $1"
      ;;
  esac
  shift
done

if [ -z "$SOURCE_REPO" ]; then
  SOURCE_REPO="$(detect_main_worktree)"
fi

[ -n "$SOURCE_REPO" ] || die "could not detect the original worktree"
[ -d "$SOURCE_REPO" ] || die "source repo not found: $SOURCE_REPO"

ENV_FILES=(
  ".env"
  ".env.local"
  ".env.development.local"
  ".env.production.local"
  ".env.test.local"
  "apps/web/.env"
  "apps/web/.env.local"
  "apps/web/.env.development.local"
  "apps/web/.env.production.local"
  "apps/web/.env.test.local"
)

copied_count=0
skipped_count=0

for rel_path in "${ENV_FILES[@]}"; do
  src="$SOURCE_REPO/$rel_path"
  dst="$PROJECT_DIR/$rel_path"

  if [ ! -f "$src" ]; then
    continue
  fi

  if [ "$src" = "$dst" ]; then
    continue
  fi

  mkdir -p "$(dirname "$dst")"

  if [ -e "$dst" ] && [ "$FORCE_COPY" -ne 1 ]; then
    log "skip existing: $rel_path"
    skipped_count=$((skipped_count + 1))
    continue
  fi

  cp "$src" "$dst"
  log "copied: $rel_path"
  copied_count=$((copied_count + 1))
done

if [ "$RUN_INSTALL" -eq 1 ]; then
  log "running pnpm install"
  (cd "$PROJECT_DIR" && pnpm install)
fi

log "bootstrap complete"
log "env copied: $copied_count"
log "env skipped: $skipped_count"
log "next: pnpm dev"
