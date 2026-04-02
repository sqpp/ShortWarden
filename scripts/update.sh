#!/usr/bin/env sh
set -eu

docker compose pull
docker compose up -d --remove-orphans
docker image prune -f

echo "Update completed."
