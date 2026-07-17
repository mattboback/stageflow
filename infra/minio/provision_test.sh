#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT

mkdir -p "$temp_dir/bin" "$temp_dir/imports"

cat >"$temp_dir/bin/mc" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

: "${TEST_MC_LOG:?}"
: "${TEST_IMPORT_DIR:?}"

printf '%q ' "$@" >>"$TEST_MC_LOG"
printf '\n' >>"$TEST_MC_LOG"

if [[ "${1:-}" == "ready" ]]; then
	exit 0
fi

if [[ "${1:-}" == "ls" ]]; then
	exit 0
fi

if [[ "${1:-}" == "ilm" && "${2:-}" == "rule" && "${3:-}" == "import" ]]; then
	target="${4:?missing lifecycle target}"
	bucket="${target##*/}"
	tee "$TEST_IMPORT_DIR/${bucket}.json" >/dev/null
fi

if [[ "${1:-}" == "admin" && "${2:-}" == "policy" && "${3:-}" == "create" ]]; then
	cp "${6:?missing policy file}" "$TEST_IMPORT_DIR/app-policy.json"
fi
EOF
chmod +x "$temp_dir/bin/mc"

cat >"$temp_dir/bin/fake-podman" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

: "${TEST_PODMAN_LOG:?}"
printf '%s\n' "$@" >"$TEST_PODMAN_LOG"
EOF
chmod +x "$temp_dir/bin/fake-podman"

# The upgrade preparation command is commonly run alongside an existing .env.
# Explicit command-line values must win so a file default cannot accidentally
# enable short artifact lifecycles before legacy baselines are copied.
mkdir -p "$temp_dir/launcher"
cat >"$temp_dir/launcher/.env" <<'EOF'
MINIO_ROOT_USER=file-root
MINIO_ROOT_PASSWORD=file-secret
MINIO_STAGING_RETENTION_DAYS=1
MINIO_ARTIFACT_RETENTION_DAYS=1
MINIO_APPLY_LIFECYCLES=true
EOF
(
	cd "$temp_dir/launcher"
	TEST_PODMAN_LOG="$temp_dir/podman.log" \
		PODMAN="$temp_dir/bin/fake-podman" \
		MINIO_STAGING_RETENTION_DAYS=7 \
		MINIO_ARTIFACT_RETENTION_DAYS=9 \
		MINIO_APPLY_LIFECYCLES=false \
		"$SCRIPT_DIR/init-buckets.sh"
)
grep -Fxq 'MINIO_ROOT_USER=file-root' "$temp_dir/podman.log"
grep -Fxq 'MINIO_STAGING_RETENTION_DAYS=7' "$temp_dir/podman.log"
grep -Fxq 'MINIO_ARTIFACT_RETENTION_DAYS=9' "$temp_dir/podman.log"
grep -Fxq 'MINIO_APPLY_LIFECYCLES=false' "$temp_dir/podman.log"
if grep -Fxq 'MINIO_APPLY_LIFECYCLES=true' "$temp_dir/podman.log"; then
	echo "explicit lifecycle override was replaced by .env" >&2
	exit 1
fi

run_provision() {
	PATH="$temp_dir/bin:$PATH" \
	TEST_MC_LOG="$temp_dir/mc.log" \
	TEST_IMPORT_DIR="$temp_dir/imports" \
	MINIO_ROOT_USER=root \
	MINIO_ROOT_PASSWORD=secret \
	MINIO_ACCESS_KEY='' \
	MINIO_SECRET_KEY='' \
	MINIO_STAGING_RETENTION_DAYS=2 \
	MINIO_ARTIFACT_RETENTION_DAYS=3 \
	"$SCRIPT_DIR/provision.sh" >/dev/null
}

# Re-running provisioning must replace the same deterministic lifecycle
# documents, not append a growing collection of rules.
run_provision
run_provision

[[ "$(grep -c 'ilm rule import stageflow/scanner-staging' "$temp_dir/mc.log")" -eq 2 ]]
[[ "$(grep -c 'ilm rule import stageflow/scanner-artifacts' "$temp_dir/mc.log")" -eq 2 ]]
grep -Fq '"ID": "stageflow-staging-retention"' "$temp_dir/imports/scanner-staging.json"
grep -Fq '"Expiration": { "Days": 2 }' "$temp_dir/imports/scanner-staging.json"
grep -Fq '"ID": "stageflow-artifact-retention"' "$temp_dir/imports/scanner-artifacts.json"
grep -Fq '"Expiration": { "Days": 3 }' "$temp_dir/imports/scanner-artifacts.json"
grep -Fq 'anonymous set none stageflow/scanner-baselines' "$temp_dir/mc.log"
[[ ! -e "$temp_dir/imports/scanner-baselines.json" ]]

PATH="$temp_dir/bin:$PATH" \
TEST_MC_LOG="$temp_dir/policy.log" \
TEST_IMPORT_DIR="$temp_dir/imports" \
MINIO_ROOT_USER=root \
MINIO_ROOT_PASSWORD=secret \
MINIO_ACCESS_KEY=stageflow \
MINIO_SECRET_KEY=app-secret \
MINIO_STAGING_RETENTION_DAYS=2 \
MINIO_ARTIFACT_RETENTION_DAYS=3 \
"$SCRIPT_DIR/provision.sh" >/dev/null
grep -Fq '"arn:aws:s3:::scanner-baselines"' "$temp_dir/imports/app-policy.json"
grep -Fq '"arn:aws:s3:::scanner-baselines/*"' "$temp_dir/imports/app-policy.json"

# Migration preparation must create buckets and refresh the app policy without
# changing existing lifecycle rules. This lets operators preserve legacy
# baseline reports before enabling short artifact retention.
: >"$temp_dir/prepare.log"
rm -f "$temp_dir/imports/scanner-staging.json" "$temp_dir/imports/scanner-artifacts.json"
PATH="$temp_dir/bin:$PATH" \
TEST_MC_LOG="$temp_dir/prepare.log" \
TEST_IMPORT_DIR="$temp_dir/imports" \
MINIO_ROOT_USER=root \
MINIO_ROOT_PASSWORD=secret \
MINIO_ACCESS_KEY=stageflow \
MINIO_SECRET_KEY=app-secret \
MINIO_APPLY_LIFECYCLES=false \
"$SCRIPT_DIR/provision.sh" >/dev/null
grep -Fq 'anonymous set none stageflow/scanner-baselines' "$temp_dir/prepare.log"
grep -Fq '"arn:aws:s3:::scanner-baselines/*"' "$temp_dir/imports/app-policy.json"
if grep -Fq 'ilm rule import' "$temp_dir/prepare.log"; then
	echo "prepare-only provisioning must not import lifecycle rules" >&2
	exit 1
fi
[[ ! -e "$temp_dir/imports/scanner-staging.json" ]]
[[ ! -e "$temp_dir/imports/scanner-artifacts.json" ]]

if PATH="$temp_dir/bin:$PATH" \
	TEST_MC_LOG="$temp_dir/invalid.log" \
	TEST_IMPORT_DIR="$temp_dir/imports" \
	MINIO_ROOT_USER=root \
	MINIO_ROOT_PASSWORD=secret \
	MINIO_STAGING_RETENTION_DAYS=0 \
	"$SCRIPT_DIR/provision.sh" >/dev/null 2>&1; then
	echo "expected zero-day retention to be rejected" >&2
	exit 1
fi

if [[ -s "$temp_dir/invalid.log" ]]; then
	echo "retention validation must fail before contacting MinIO" >&2
	exit 1
fi

echo "MinIO provisioning tests passed"
