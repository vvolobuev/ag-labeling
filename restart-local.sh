#!/bin/sh

set -e
ROOT="$(cd "$(dirname "$0")" && pwd)"
MODE="${1:-}"

PORT="$(grep '^SERVER_PORT=' "$ROOT/backend/.env" 2>/dev/null | cut -d= -f2)"
PORT="${PORT:-8095}"

storage_export() {
  STORAGE_ABS="$ROOT/backend/storage"
  mkdir -p "$STORAGE_ABS"
  export STORAGE_ROOT="$STORAGE_ABS"
  TMP_ABS="$ROOT/backend/storage/tmp"
  mkdir -p "$TMP_ABS"
  export TMPDIR="$TMP_ABS"
}

stop_listeners() {
  for p in "$PORT" 5173 8081; do
    fuser -k "${p}/tcp" >/dev/null 2>&1 || true
  done
  sleep 1
}

storage_export
stop_listeners

if [ "$MODE" = "--bg" ]; then
  echo "Фон: API :$PORT, Vite :5173, file-server :8081 | STORAGE_ROOT=$STORAGE_ROOT | TMPDIR=$TMPDIR"
  (cd "$ROOT/backend" && go run .) >> /tmp/alpha-guard-backend.log 2>&1 &
  (cd "$ROOT/file-server" && go run .) >> /tmp/alpha-guard-fileserver.log 2>&1 &
  (cd "$ROOT/frontend" && npm run dev) >> /tmp/alpha-guard-frontend.log 2>&1 &
  sleep 4
  echo "Логи: /tmp/alpha-guard-backend.log | fileserver | frontend"
  ss -tlnp 2>/dev/null | grep -E ":${PORT}\\b|:5173\\b|:8081\\b" || true
  exit 0
fi

exec "$ROOT/run-local.sh"
