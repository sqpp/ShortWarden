#!/usr/bin/env sh
set -eu

PROJECT="${SHORTWARDEN_COMPOSE_PROJECT_NAME:-shortwarden}"
if [ -n "${SHORTWARDEN_COMPOSE_FILE:-}" ]; then
  CF="$SHORTWARDEN_COMPOSE_FILE"
elif [ -f docker-compose.nginx.yml ]; then
  CF=docker-compose.nginx.yml
else
  CF=docker-compose.yml
fi

dc() {
  docker compose -f "$CF" -p "$PROJECT" "$@"
}

dc pull api
if dc config --services 2>/dev/null | grep -qx screenshot; then
  dc build screenshot
  dc up -d --no-deps --force-recreate api screenshot
else
  dc up -d --no-deps --force-recreate api
fi
dc restart nginx
docker image prune -f

echo "Update completed."
