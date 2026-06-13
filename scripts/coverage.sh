#!/usr/bin/env bash
set -euo pipefail

packages="$(go list ./internal/... ./pkg/... 2>&1 || true)"
if [[ -z "${packages}" ]]; then
  echo "no coverage packages found"
  exit 0
fi

go test -coverprofile=coverage.out ${packages}

total="$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%", "", $3); print $3}')"
minimum="${COVERAGE_MIN:-80}"

echo "total coverage: ${total}%"

awk -v total="${total}" -v minimum="${minimum}" 'BEGIN { exit !(total + 0 >= minimum + 0) }'

