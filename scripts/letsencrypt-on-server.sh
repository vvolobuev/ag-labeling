#!/bin/sh
set -e
export DEBIAN_FRONTEND=noninteractive

APP_DIR="${APP_DIR:-/opt/alpha-guard-ai}"
cd "$APP_DIR"

apt-get update -qq
apt-get install -y -qq certbot

mkdir -p /var/www/certbot

docker compose down 2>/dev/null || true

if ! certbot certonly --standalone \
  --non-interactive --agree-tos --register-unsafely-without-email \
  -d alpha-guard.online -d www.alpha-guard.online; then
  certbot certonly --standalone \
    --non-interactive --agree-tos --register-unsafely-without-email \
    -d alpha-guard.online
fi

if grep -q '^PUBLIC_APP_URL=' backend/.env 2>/dev/null; then
  sed -i 's|^PUBLIC_APP_URL=.*|PUBLIC_APP_URL=https://alpha-guard.online|' backend/.env
else
  echo 'PUBLIC_APP_URL=https://alpha-guard.online' >> backend/.env
fi

export DOCKER_BUILDKIT="${DOCKER_BUILDKIT:-1}"
docker compose build --progress=plain
docker compose up -d --force-recreate

echo "TLS OK. Open https://alpha-guard.online"
echo "Renewal: certbot renew --dry-run (add cron with docker compose restart if needed)"
