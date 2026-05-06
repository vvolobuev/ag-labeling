#!/bin/sh
# На VPS под root: удалить развёртывание ALPHA GUARD AI (compose, файлы, тома приложения).
# Не трогает контейнер postgres-db и вашу БД.

set +e
echo ">>> stopping compose (if present)"
if [ -f /opt/alpha-guard-ai/docker-compose.yml ]; then
  cd /opt/alpha-guard-ai && docker compose down --volumes --remove-orphans 2>/dev/null
fi

echo ">>> remove docker images built for this project"
for id in $(docker images -q 2>/dev/null); do
  tags="$(docker inspect -f '{{range .RepoTags}}{{.}} {{end}}' "$id" 2>/dev/null)"
  echo "$tags" | grep -q alpha-guard-ai && docker rmi -f "$id" 2>/dev/null
done

echo ">>> remove compose-named volume"
docker volume rm alpha-guard-ai_alphaguard_storage 2>/dev/null
docker volume ls -q | grep -i alphaguard_storage | while read -r vol; do
  docker volume rm "$vol" 2>/dev/null
done

echo ">>> remove tree and temp uploads"
rm -rf /opt/alpha-guard-ai
rm -f /tmp/alpha-guard-ai-deploy.tgz /tmp/deploy-on-server.sh /tmp/ag-compose.log \
  /tmp/server-finish-deploy.sh /tmp/ag-build.log /tmp/ag-deploy.log /tmp/ag-deploy.pid

echo ">>> done. Remaining containers:"
docker ps -a --format '{{.Names}} {{.Image}}' | head -20
