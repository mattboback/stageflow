#!/usr/bin/env bash
# stdin: plain pg_dump of StageFlow v2. stdout: sanitized plain pg_dump.
# The raw snapshot exists only in a private temporary PostgreSQL data directory.
set -euo pipefail
umask 077
sanitizer="${STAGEFLOW_BACKUP_SANITIZER:?STAGEFLOW_BACKUP_SANITIZER must name the reviewed installed binary}"
[[ -x "$sanitizer" ]] || { echo 'Backup sanitizer is not executable.' >&2; exit 1; }
for tool in initdb pg_ctl createdb psql pg_dump; do command -v "$tool" >/dev/null; done
workspace="$(mktemp -d "${TMPDIR:-/var/tmp}/stageflow-backup.XXXXXX")"
started=0
cleanup() {
  if (( started )); then pg_ctl -D "$workspace/data" -m immediate -w stop >/dev/null 2>&1 || status=1; fi
  rm -rf "$workspace"
}
trap 'status=$?; cleanup; exit "$status"' EXIT
mkdir -m 0700 "$workspace/socket"
initdb -D "$workspace/data" -U stageflow_backup -A trust --no-instructions > "$workspace/init.log" 2>&1
started=1
pg_ctl -D "$workspace/data" -l "$workspace/server.log" -o "-c listen_addresses='' -c unix_socket_directories='$workspace/socket' -c unix_socket_permissions=0700" -w start > "$workspace/start.log" 2>&1
export PGHOST="$workspace/socket" PGPORT=5432 PGUSER=stageflow_backup PGDATABASE=stageflow_backup
unset PGSERVICE PGSERVICEFILE PGOPTIONS PGPASSWORD PGHOSTADDR
createdb
# Prevent restore output/error text from echoing literal credentials into logs.
psql -X -v ON_ERROR_STOP=1 > "$workspace/restore.log" 2>&1 || { echo 'Restoring the isolated backup snapshot failed.' >&2; exit 1; }
"$sanitizer" "$workspace/socket"
pg_dump --clean --if-exists --no-owner --no-privileges
