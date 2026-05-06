#!/bin/sh
set -e
export STORAGE_ROOT="${STORAGE_ROOT:-/app/storage}"
mkdir -p "$STORAGE_ROOT"
cd /app
./backend/main &
sleep 2
exec nginx -g 'daemon off;'
