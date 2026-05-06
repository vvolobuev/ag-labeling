#!/bin/sh
set -e
TARBALL="${TARBALL:-/tmp/alpha-guard-ai-deploy.tgz}"
APP_DIR="${APP_DIR:-/opt/alpha-guard-ai}"

if [ ! -f "$APP_DIR/docker-compose.yml" ]; then
  if [ ! -f "$TARBALL" ]; then
    echo "Need $TARBALL or existing $APP_DIR"
    exit 1
  fi
  mkdir -p /opt
  rm -rf "$APP_DIR"
  tar xzf "$TARBALL" -C /opt
fi

cd "$APP_DIR"
test -f backend/.env || { echo "missing backend/.env"; exit 1; }

sed -i 's/^DB_HOST=.*/DB_HOST=127.0.0.1/' backend/.env
if grep -q '^PUBLIC_APP_URL=' backend/.env; then
  sed -i "s|^PUBLIC_APP_URL=.*|PUBLIC_APP_URL=https://alpha-guard.online|" backend/.env
else
  echo 'PUBLIC_APP_URL=https://alpha-guard.online' >> backend/.env
fi

command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q active && {
  ufw allow 80/tcp comment alpha-guard-http 2>/dev/null || true
  ufw allow 443/tcp comment alpha-guard-https 2>/dev/null || true
}

docker compose version
LOG=/tmp/ag-compose.log
: >"$LOG"

export DOCKER_BUILDKIT="${DOCKER_BUILDKIT:-1}"

docker compose down --remove-orphans 2>/dev/null || true

docker compose build --progress=plain 2>&1 | tee -a "$LOG"

docker compose up -d --force-recreate 2>&1 | tee -a "$LOG"
sleep 3
docker compose ps -a || true

curl -sS -o /dev/null -w "HTTP localhost %s\n" http://127.0.0.1/ || echo "curl failed"

docker ps --filter name=alphaguard -a 2>/dev/null || docker ps | head || true

echo "Done."
