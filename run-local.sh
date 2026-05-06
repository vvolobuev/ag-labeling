#!/bin/sh
# Локально: API + файловый сервер + Vite.
# Изображения датасета пишутся ТОЛЬКО НА ЭТОМ ПК (STORAGE_ROOT), не на хост где PostgreSQL.
# DB_HOST может указывать на удалённый сервер — в БД лежат только метаданные/разметка.
set -e
ROOT="$(cd "$(dirname "$0")" && pwd)"

STORAGE_ABS="$ROOT/backend/storage"
mkdir -p "$STORAGE_ABS"
export STORAGE_ROOT="$STORAGE_ABS"
TMP_ABS="$ROOT/backend/storage/tmp"
mkdir -p "$TMP_ABS"
export TMPDIR="$TMP_ABS"

DBH="$(grep '^DB_HOST=' "$ROOT/backend/.env" | cut -d= -f2)"
DBP="$(grep '^DB_PORT=' "$ROOT/backend/.env" | cut -d= -f2)"
echo "Файлы датасетов на этом ПК: $STORAGE_ABS"
echo "Временные multipart-файлы: $TMPDIR"
echo "БД (метаданные): ${DBH}:${DBP} — с ПК должен быть доступен порт 5432 (firewall)."
echo "Откройте: http://localhost:5173"

(cd "$ROOT/backend" && go run .) &
BPID=$!
(cd "$ROOT/file-server" && go run .) &
FPID=$!
cleanup() { kill "$BPID" "$FPID" 2>/dev/null; }
trap cleanup INT TERM EXIT

cd "$ROOT/frontend"
npm run dev
cleanup
