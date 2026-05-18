#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -f "${root_dir}/.env" ]]; then
  set -a
  source "${root_dir}/.env"
  set +a
fi

postgres_user="${POSTGRES_USER:-postgres}"
postgres_password="${POSTGRES_PASSWORD:-postgres}"
postgres_db="${POSTGRES_DB:-task_platform}"
postgres_host="${POSTGRES_HOST:-127.0.0.1}"
postgres_port="${POSTGRES_PORT:-5433}"
migrate_image="${MIGRATE_IMAGE:-migrate/migrate:v4.18.3}"

wait_for_postgres() {
  local max_attempts=${MIGRATE_WAIT_ATTEMPTS:-30}
  local attempt=1
  echo -n "waiting for postgres ${postgres_host}:${postgres_port}"
  while [ $attempt -le $max_attempts ]; do
    if timeout 1 bash -c "echo > /dev/tcp/${postgres_host}/${postgres_port}" 2>/dev/null; then
      echo " ready"
      return 0
    fi
    echo -n "."
    sleep 2
    attempt=$((attempt + 1))
  done
  echo " timeout"
  return 1
}

wait_for_postgres || echo "WARNING: postgres not reachable, migration may fail"

run_schema_migration() {
  local schema="$1"
  local dir="${root_dir}/migrations/${schema}"

  if [[ ! -d "${dir}" ]]; then
    echo "skip ${schema}: missing migration directory"
    return
  fi

  docker run --rm \
    --network host \
    -v "${dir}:/migrations" \
    "${migrate_image}" \
    -path=/migrations \
    -database "postgres://${postgres_user}:${postgres_password}@${postgres_host}:${postgres_port}/${postgres_db}?sslmode=disable&x-migrations-table=schema_migrations_${schema}" \
    up
}

run_schema_migration "user_svc"
run_schema_migration "task_svc"

