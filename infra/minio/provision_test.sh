#!/usr/bin/env bash
set -euo pipefail

# The invoking Just process may have loaded its checkout's .env.
unset ENV_FILE MINIO_ENDPOINT MINIO_ROOT_USER MINIO_ROOT_PASSWORD \
	MINIO_ACCESS_KEY MINIO_SECRET_KEY MINIO_STAGING_RETENTION_DAYS \
	MINIO_ARTIFACT_RETENTION_DAYS MINIO_APP_POLICY MINIO_ALIAS MC_IMAGE MC_NETWORK

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

# Explicit command-line retention values must win over an existing .env.
mkdir -p "$temp_dir/launcher"
cat >"$temp_dir/launcher/.env" <<'EOF'
MINIO_ROOT_USER=file-root
MINIO_ROOT_PASSWORD=file-secret
MINIO_STAGING_RETENTION_DAYS=1
MINIO_ARTIFACT_RETENTION_DAYS=1
EOF
(
	cd "$temp_dir/launcher"
	TEST_PODMAN_LOG="$temp_dir/podman.log" \
		PODMAN="$temp_dir/bin/fake-podman" \
		MINIO_STAGING_RETENTION_DAYS=7 \
		MINIO_ARTIFACT_RETENTION_DAYS=9 \
		"$SCRIPT_DIR/init-buckets.sh"
)
grep -Fxq 'MINIO_ROOT_USER=file-root' "$temp_dir/podman.log"
grep -Fxq 'MINIO_STAGING_RETENTION_DAYS=7' "$temp_dir/podman.log"
grep -Fxq 'MINIO_ARTIFACT_RETENTION_DAYS=9' "$temp_dir/podman.log"

# A selected external file replaces .env, including optional values it omits.
cat >>"$temp_dir/launcher/.env" <<'EOF'
MINIO_APP_POLICY=local-only-policy
EOF
cat >"$temp_dir/external.env" <<'EOF'
MINIO_ROOT_USER=external-root
MINIO_ROOT_PASSWORD=external-secret
MINIO_STAGING_RETENTION_DAYS=4
EOF
(
	cd "$temp_dir/launcher"
	ENV_FILE="$temp_dir/external.env" \
		TEST_PODMAN_LOG="$temp_dir/external-podman.log" \
		PODMAN="$temp_dir/bin/fake-podman" \
		MINIO_ARTIFACT_RETENTION_DAYS=8 \
		"$SCRIPT_DIR/init-buckets.sh"
)
grep -Fxq 'MINIO_ROOT_USER=external-root' "$temp_dir/external-podman.log"
grep -Fxq 'MINIO_ROOT_PASSWORD=external-secret' "$temp_dir/external-podman.log"
grep -Fxq 'MINIO_STAGING_RETENTION_DAYS=4' "$temp_dir/external-podman.log"
grep -Fxq 'MINIO_ARTIFACT_RETENTION_DAYS=8' "$temp_dir/external-podman.log"
grep -Fxq 'MINIO_APP_POLICY=stageflow-app' "$temp_dir/external-podman.log"

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
[[ "$(grep -c 'anonymous set none' "$temp_dir/mc.log")" -eq 4 ]]

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
grep -Fq '"arn:aws:s3:::scanner-staging"' "$temp_dir/imports/app-policy.json"
grep -Fq '"arn:aws:s3:::scanner-artifacts/*"' "$temp_dir/imports/app-policy.json"
[[ "$(grep -c 'arn:aws:s3:::' "$temp_dir/imports/app-policy.json")" -eq 4 ]]

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
