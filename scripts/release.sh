#!/usr/bin/env sh
set -eu

if [ "${1-}" = "" ]; then
  echo "Usage: ./scripts/release.sh <version> [image]"
  echo "Example: ./scripts/release.sh 1.0.5 sqpp/shortwarden"
  exit 1
fi

VERSION="$1"
IMAGE="${2:-sqpp/shortwarden}"

if command -v git >/dev/null 2>&1; then
  GIT_SHA="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
else
  GIT_SHA="unknown"
fi
BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

echo "Building ${IMAGE}:${VERSION}"
docker build -f Dockerfile.api \
  --build-arg APP_VERSION="${VERSION}" \
  --build-arg GIT_SHA="${GIT_SHA}" \
  --build-arg BUILD_TIME="${BUILD_TIME}" \
  -t "${IMAGE}:${VERSION}" \
  -t "${IMAGE}:latest" \
  .

echo "Pushing ${IMAGE}:${VERSION}"
docker push "${IMAGE}:${VERSION}"
echo "Pushing ${IMAGE}:latest"
docker push "${IMAGE}:latest"

echo "Release completed: ${IMAGE}:${VERSION}"
