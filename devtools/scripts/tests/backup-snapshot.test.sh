#!/usr/bin/env bash
# Real source -> isolated sanitizer -> SQL dump -> restored database regression.
set -euo pipefail
umask 077
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
work="$(mktemp -d "${TMPDIR:-/var/tmp}/stageflow-backup-test.XXXXXX")"
cleanup() { pg_ctl -D "$work/data" -m immediate -w stop >/dev/null 2>&1 || true; rm -rf "$work"; }
trap cleanup EXIT
mkdir -m 0700 "$work/socket"
initdb -D "$work/data" -U stageflow_test -A trust --no-instructions > "$work/init.log" 2>&1
pg_ctl -D "$work/data" -l "$work/server.log" -o "-c listen_addresses='' -c unix_socket_directories='$work/socket'" -w start > "$work/start.log" 2>&1
export PGHOST="$work/socket" PGPORT=5432 PGUSER=stageflow_test PGDATABASE=source
unset PGSERVICE PGSERVICEFILE PGOPTIONS PGPASSWORD PGHOSTADDR
createdb
psql -X -v ON_ERROR_STOP=1 -f "$root/internal/jobstore/schema.sql" > /dev/null
psql -X -v ON_ERROR_STOP=1 > /dev/null <<'SQL'
INSERT INTO jobs (id,state,input_type,config_json,urls,error,error_details,input_path) VALUES
('form','SCANNING','urls','{"auth":{"mode":"form","steps":[{"type":"fill","value":{"literal":"backup password!"}},{"type":"select","value":{"literal":"tenant-canary"}}]},"pages":[{"inputValues":{"#otp":"input-canary"}}],"safe":"ordinary metadata","scanner_configs":{"axe":{"nested":[{"backup password!":"first collision payload","backup%20password!":"second collision payload","safe":"ordinary metadata"},{"backup+password%21":"safe value"}]}}}', '["https://example.com/?password=backup%20password!"]','backup password!','tenant-canary','input-canary'),
('storage','PENDING','urls','{"auth":{"mode":"storage_state","content_b64":"eyJjb29raWVzIjpbeyJ2YWx1ZSI6ImNvb2tpZS1jYW5hcnkifV0sIm9yaWdpbnMiOlt7ImxvY2FsU3RvcmFnZSI6W3sidmFsdWUiOiJsb2NhbHN0b3JlLWNhbmFyeSJ9XX1dfQ=="}}','[]','','',''),
('done','DONE','urls','{"scanner_configs":{"axe":{"cookie-canary":"first cookie payload","localstore-canary":"second cookie payload"}}}','[]','historical benign failure','','');
INSERT INTO job_scanners (job_id,scanner,state,error,results_path) VALUES
('form','axe','running','backup+password%21','input-canary/report.json'),
('done','axe','success','cookie-canary and localstore-canary','jobs/done/report.json');
SQL
cd "$root"
go build -o "$work/sanitize-backup" ./devtools/cmd/sanitize-backup
export STAGEFLOW_BACKUP_SANITIZER="$work/sanitize-backup"
# The command rejects arbitrary sockets before opening a database.
if "$STAGEFLOW_BACKUP_SANITIZER" "$work/socket" > "$work/guard.out" 2> "$work/guard.log"; then echo 'Unrestricted socket accepted' >&2; exit 1; fi
pg_dump --clean --if-exists --no-owner --no-privileges > "$work/raw.sql"
bash devtools/scripts/backup-snapshot.sh < "$work/raw.sql" > "$work/sanitized.sql"
python3 - "$work/sanitized.sql" <<'PY'
import sys
dump = open(sys.argv[1]).read()
for secret in ['backup password!', 'backup%20password!', 'backup+password%21', 'tenant-canary', 'input-canary', 'cookie-canary', 'localstore-canary', 'eyJjb29raWVzIjpbeyJ2YWx1ZSI6ImNvb2tpZS1jYW5hcnkifV0sIm9yaWdpbnMiOlt7ImxvY2FsU3RvcmFnZSI6W3sidmFsdWUiOiJsb2NhbHN0b3JlLWNhbmFyeSJ9XX1dfQ==']:
    assert secret not in dump, f'credential survives sanitized dump: {secret}'
assert 'ordinary metadata' in dump
for discarded in ['first collision payload', 'second collision payload', 'first cookie payload', 'second cookie payload']:
    assert discarded not in dump
PY
createdb restored
PGDATABASE=restored psql -X -v ON_ERROR_STOP=1 < "$work/sanitized.sql" > /dev/null
[[ "$(PGDATABASE=restored psql -X -Atc "SELECT count(*) FROM jobs WHERE id IN ('form','storage') AND state='FAILED' AND completed_at IS NOT NULL AND NOT cleanup_pending")" == 2 ]]
[[ "$(PGDATABASE=restored psql -X -Atc "SELECT state FROM jobs WHERE id='done'")" == DONE ]]
[[ "$(PGDATABASE=restored psql -X -Atc "SELECT config_json#>>'{scanner_configs,axe,nested,0,[REDACTED]}' FROM jobs WHERE id='form'")" == '[REDACTED]' ]]
[[ "$(PGDATABASE=restored psql -X -Atc "SELECT config_json#>>'{scanner_configs,axe,[REDACTED]}' FROM jobs WHERE id='done'")" == '[REDACTED]' ]]
[[ "$(PGDATABASE=restored psql -X -Atc "SELECT error FROM jobs WHERE id='done'")" == 'historical benign failure' ]]
[[ "$(psql -X -Atc "SELECT state FROM jobs WHERE id='form'")" == SCANNING ]]
[[ "$(psql -X -Atc "SELECT config_json->'auth'->>'mode' FROM jobs WHERE id='form'")" == form ]]
# Unreadable configuration and schema additions fail closed before producing SQL.
psql -X -v ON_ERROR_STOP=1 -c "UPDATE jobs SET config_json='[]' WHERE id='form'" > /dev/null
pg_dump --clean --if-exists --no-owner --no-privileges > "$work/malformed.sql"
if bash devtools/scripts/backup-snapshot.sh < "$work/malformed.sql" > "$work/rejected.sql" 2> "$work/rejected.log"; then echo 'Malformed configuration accepted' >&2; exit 1; fi
[[ ! -s "$work/rejected.sql" ]]
psql -X -v ON_ERROR_STOP=1 -c "UPDATE jobs SET config_json='{}'; CREATE TABLE unexpected_secret(payload text)" > /dev/null
pg_dump --clean --if-exists --no-owner --no-privileges > "$work/drift.sql"
if bash devtools/scripts/backup-snapshot.sh < "$work/drift.sql" > "$work/rejected.sql" 2> "$work/rejected.log"; then echo 'Unknown table accepted' >&2; exit 1; fi
[[ ! -s "$work/rejected.sql" ]]
if STAGEFLOW_BACKUP_SANITIZER=/bin/false bash devtools/scripts/backup-snapshot.sh < "$work/raw.sql" > "$work/rejected.sql" 2> "$work/rejected.log"; then echo 'Sanitizer failure accepted' >&2; exit 1; fi
[[ ! -s "$work/rejected.sql" ]]
echo 'backup-snapshot test passed (restorable, source unchanged, fail closed)'
