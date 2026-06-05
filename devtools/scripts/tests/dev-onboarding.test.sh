#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
REAL_JUST="${JUST:-$(command -v just)}"

failures=0

assert_contains() {
	local file="$1"
	local expected="$2"
	local message="$3"

	if ! grep -Fq -- "$expected" "$file"; then
		echo "FAIL: ${message}" >&2
		echo "Expected to find: ${expected}" >&2
		failures=$((failures + 1))
	fi
}

assert_line_before() {
	local file="$1"
	local first="$2"
	local second="$3"
	local message="$4"
	local first_line
	local second_line

	first_line="$(grep -Fnx -- "$first" "$file" | head -n1 | cut -d: -f1 || true)"
	second_line="$(grep -Fnx -- "$second" "$file" | head -n1 | cut -d: -f1 || true)"

	if [[ -z "$first_line" || -z "$second_line" || "$first_line" -ge "$second_line" ]]; then
		echo "FAIL: ${message}" >&2
		echo "Expected '${first}' to appear before '${second}'" >&2
		failures=$((failures + 1))
	fi
}

write_fake_podman() {
	local path="$1"

	cat >"$path" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

: "${STAGEFLOW_TEST_LOG:?STAGEFLOW_TEST_LOG must be set}"

printf 'podman' >>"$STAGEFLOW_TEST_LOG"
printf ' %q' "$@" >>"$STAGEFLOW_TEST_LOG"
printf '\n' >>"$STAGEFLOW_TEST_LOG"

case "${1:-}" in
	network)
		case "${2:-}" in
			inspect)
				exit 1
				;;
			create)
				exit 0
				;;
		esac
		;;
	image)
		if [[ "${2:-}" == "exists" ]]; then
			exit 0
		fi
		;;
	compose)
		printf 'compose-env STAGEFLOW_NETWORK_NAME=%s\n' "${STAGEFLOW_NETWORK_NAME:-}" >>"$STAGEFLOW_TEST_LOG"
		exit 0
		;;
	info)
		exit 0
		;;
esac

exit 0
EOF

	chmod +x "$path"
}

run_just_with_fake_podman() {
	local temp_dir="$1"
	shift

	STAGEFLOW_TEST_LOG="${temp_dir}/podman.log" \
	PODMAN="${temp_dir}/podman" \
	"$REAL_JUST" --justfile "$REPO_ROOT/justfile" --working-directory "$REPO_ROOT" "$@"
}

run_network_default_case() {
	local temp_dir="$1"
	local log_file="${temp_dir}/podman.log"

	: >"$log_file"
	run_just_with_fake_podman "$temp_dir" dev down

	assert_contains "$log_file" "podman network inspect stageflow_dev_net" "dev should inspect the project-scoped default network"
	assert_contains "$log_file" "podman network create stageflow_dev_net" "dev should create the project-scoped default network when missing"
	assert_contains "$log_file" "compose-env STAGEFLOW_NETWORK_NAME=stageflow_dev_net" "dev should export the effective default network to compose"
}

run_network_project_case() {
	local temp_dir="$1"
	local log_file="${temp_dir}/podman.log"

	: >"$log_file"
	COMPOSE_PROJECT_NAME=stageflow_fresh_audit run_just_with_fake_podman "$temp_dir" dev up dev http://127.0.0.1:9000 minio

	assert_contains "$log_file" "podman network inspect stageflow_fresh_audit_net" "dev should derive the network from COMPOSE_PROJECT_NAME"
	assert_contains "$log_file" "compose-env STAGEFLOW_NETWORK_NAME=stageflow_fresh_audit_net" "dev should export the project-derived network"
	assert_contains "$log_file" "podman compose -p stageflow_fresh_audit" "dev should use COMPOSE_PROJECT_NAME as compose project"
	assert_contains "$log_file" "up -d minio" "dev should pass an explicit service list through to compose up"
}

run_network_override_case() {
	local temp_dir="$1"
	local log_file="${temp_dir}/podman.log"

	: >"$log_file"
	COMPOSE_PROJECT_NAME=stageflow_fresh_audit STAGEFLOW_NETWORK_NAME=custom_net run_just_with_fake_podman "$temp_dir" dev down

	assert_contains "$log_file" "podman network inspect custom_net" "STAGEFLOW_NETWORK_NAME should override the project-derived network"
	assert_contains "$log_file" "compose-env STAGEFLOW_NETWORK_NAME=custom_net" "dev should export the explicit network override"
}

run_demo_order_case() {
	local justfile="$REPO_ROOT/justfile"

	assert_line_before "$justfile" \
		"    just dev up dev http://127.0.0.1:9000 minio" \
		"    just dev init" \
		"demo should start MinIO before initializing buckets"

	assert_line_before "$justfile" \
		"    just dev init" \
		"    wait_for_http \"http://127.0.0.1:8080/healthz\" \"Platform API\" 120" \
		"demo should initialize MinIO before waiting for Platform API health"

	assert_line_before "$justfile" \
		"    just dev init" \
		"    just dev up" \
		"demo should start the full stack after MinIO initialization"
}

main() {
	local temp_dir
	temp_dir="$(mktemp -d)"
	trap 'rm -rf "${temp_dir:-}"' EXIT

	write_fake_podman "${temp_dir}/podman"

	run_network_default_case "$temp_dir"
	run_network_project_case "$temp_dir"
	run_network_override_case "$temp_dir"
	run_demo_order_case

	if [[ "$failures" -ne 0 ]]; then
		echo "dev onboarding tests failed: ${failures}" >&2
		exit 1
	fi

	echo "dev onboarding tests passed"
}

main "$@"
