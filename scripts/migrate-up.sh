#!/usr/bin/env bash
set -euo pipefail

POSTGRES_DSN="${1:-postgres://shortwarden:shortwarden@localhost:5432/shortwarden?sslmode=disable}"

export SHORTWARDEN_POSTGRES_DSN="$POSTGRES_DSN"

go run ./cmd/migrate

