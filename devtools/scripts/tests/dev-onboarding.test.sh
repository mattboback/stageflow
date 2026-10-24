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
		printf 'compose-env STAGEFLOW_NETWORK_NAME=%s POD_NETWORK=%s FRONTEND_PORT=%s JOBS_NETWORK=%s\n' "${STAGEFLOW_NETWORK_NAME:-}" "${POD_NETWORK-unset}" "${FRONTEND_PORT:-}" "${STAGEFLOW_JOBS_NETWORK_NAME:-}" >>"$STAGEFLOW_TEST_LOG"
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
	"$REAL_JUST" --justfile "$temp_dir/work/justfile" --working-directory "$temp_dir/work" "$@"
}

run_network_default_case() {
	local temp_dir="$1"
	local log_file="${temp_dir}/podman.log"

	: >"$log_file"
	run_just_with_fake_podman "$temp_dir" dev down

	assert_contains "$log_file" "podman network inspect stageflow_dev_net" "dev should inspect the project-scoped default network"
	assert_contains "$log_file" "podman network create stageflow_dev_net" "dev should create the project-scoped default network when missing"
	assert_contains "$log_file" "compose-env STAGEFLOW_NETWORK_NAME=stageflow_dev_net" "dev should export the effective default network to compose"
	assert_contains "$log_file" "podman network create --opt isolate=strict stageflow_dev_jobs" "dev should create the isolated job network when missing"
	assert_contains "$log_file" "JOBS_NETWORK=stageflow_dev_jobs" "dev should export the job network to compose"
}

run_network_project_case() {
	local temp_dir="$1"
	local log_file="${temp_dir}/podman.log"

	: >"$log_file"
	COMPOSE_PROJECT_NAME=stageflow_fresh_audit run_just_with_fake_podman "$temp_dir" dev up dev postgres

	assert_contains "$log_file" "podman network inspect stageflow_fresh_audit_net" "dev should derive the network from COMPOSE_PROJECT_NAME"
	assert_contains "$log_file" "compose-env STAGEFLOW_NETWORK_NAME=stageflow_fresh_audit_net" "dev should export the project-derived network"
	assert_contains "$log_file" "podman network create --opt isolate=strict stageflow_fresh_audit_jobs" "dev should derive the isolated job network from COMPOSE_PROJECT_NAME"
	assert_contains "$log_file" "podman compose -p stageflow_fresh_audit" "dev should use COMPOSE_PROJECT_NAME as compose project"
	assert_contains "$log_file" "up -d postgres" "dev should pass an explicit service list through to compose up"
}

run_network_override_case() {
	local temp_dir="$1"
	local log_file="${temp_dir}/podman.log"

	: >"$log_file"
	COMPOSE_PROJECT_NAME=stageflow_fresh_audit STAGEFLOW_NETWORK_NAME=custom_net run_just_with_fake_podman "$temp_dir" dev down

	assert_contains "$log_file" "podman network inspect custom_net" "STAGEFLOW_NETWORK_NAME should override the project-derived network"
	assert_contains "$log_file" "compose-env STAGEFLOW_NETWORK_NAME=custom_net" "dev should export the explicit network override"
}


run_env_settings_case() {
	local temp_dir="$1"
	local log_file="${temp_dir}/podman.log"
	cat >"$temp_dir/work/.env" <<'EOF'
COMPOSE_PROJECT_NAME=from_file
STAGEFLOW_NETWORK_NAME=file_net
FRONTEND_PORT=13000
POD_NETWORK=
IMAGE_TAG=isolated-test
EOF
	: >"$log_file"
	run_just_with_fake_podman "$temp_dir" dev up local
	assert_contains "$log_file" "podman compose -p from_file -f $temp_dir/work/infra/compose/podman-compose.yml --env-file $temp_dir/work/.env up -d" "local alias should use the sole compose file and project from .env"
	assert_contains "$log_file" "compose-env STAGEFLOW_NETWORK_NAME=file_net POD_NETWORK= FRONTEND_PORT=13000" "compose should receive .env network, explicit empty pod network, and port settings"
	assert_contains "$log_file" "podman image exists localhost/stageflow/extractor:isolated-test" "job image checks should use the configured image tag"

	: >"$log_file"
	COMPOSE_PROJECT_NAME=shell_project STAGEFLOW_NETWORK_NAME=shell_net run_just_with_fake_podman "$temp_dir" dev-refresh dev platform-api
	assert_contains "$log_file" "podman compose -p shell_project -f $temp_dir/work/infra/compose/podman-compose.yml" "refresh should honor shell project over .env and use the sole compose file"
	assert_contains "$log_file" "compose-env STAGEFLOW_NETWORK_NAME=shell_net" "refresh should honor shell network over .env"

	: >"$log_file"
}

run_external_env_case() {
	local temp_dir="$1"
	local log_file="${temp_dir}/podman.log"
	cat >"$temp_dir/external.env" <<'EOF'
COMPOSE_PROJECT_NAME=external_project
STAGEFLOW_NETWORK_NAME=external_net
FRONTEND_PORT=23000
EOF
	: >"$log_file"
	ENV_FILE="$temp_dir/external.env" run_just_with_fake_podman "$temp_dir" dev down
	assert_contains "$log_file" "podman compose -p external_project -f $temp_dir/work/infra/compose/podman-compose.yml --env-file $temp_dir/external.env down" "ENV_FILE should select the same file for Just and compose"
	assert_contains "$log_file" "compose-env STAGEFLOW_NETWORK_NAME=external_net POD_NETWORK=unset FRONTEND_PORT=23000" "external env should replace local .env without importing its settings"
	: >"$log_file"
	ENV_FILE="$temp_dir/external.env" FRONTEND_PORT=33000 run_just_with_fake_podman "$temp_dir" dev down
	assert_contains "$log_file" "FRONTEND_PORT=33000" "shell values should override the external env file"
}

run_demo_settings_case() {
	local temp_dir="$1"
	local log_file="${temp_dir}/podman.log"
	cat >>"$temp_dir/work/.env" <<'EOF'
PLATFORM_API_PORT=18080
VITE_API_URL=http://file-api.example
VITE_SITE_URL=http://file-site.example
PLATFORM_API_CORS_ALLOW_ORIGINS=http://file-site.example
EOF
	mkdir -p "$temp_dir/bin"
	cat >"$temp_dir/bin/just" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'just %s\n' "$*" >>"$STAGEFLOW_TEST_LOG"
printf 'demo-env API=%s SITE=%s CORS=%s\n' "${VITE_API_URL:-}" "${VITE_SITE_URL:-}" "${PLATFORM_API_CORS_ALLOW_ORIGINS:-}" >>"$STAGEFLOW_TEST_LOG"
EOF
	cat >"$temp_dir/bin/curl" <<'EOF'
#!/usr/bin/env bash
printf 'curl %s\n' "$*" >>"$STAGEFLOW_TEST_LOG"
EOF
	chmod +x "$temp_dir/bin/just" "$temp_dir/bin/curl"
	: >"$log_file"
	PATH="$temp_dir/bin:$PATH" \
	STAGEFLOW_DEMO_VITE_API_URL=http://override-api.example \
	STAGEFLOW_DEMO_VITE_SITE_URL=http://override-site.example \
	STAGEFLOW_DEMO_CORS_ALLOW_ORIGINS=http://override-site.example \
	run_just_with_fake_podman "$temp_dir" demo >"$temp_dir/demo.out"
	assert_contains "$log_file" "demo-env API=http://override-api.example SITE=http://override-site.example CORS=http://override-site.example" "demo should pass explicit overrides to builds and compose"
	assert_contains "$log_file" "curl -fsS http://127.0.0.1:18080/healthz" "demo should check the configured API port"
	assert_contains "$log_file" "curl -fsS http://127.0.0.1:13000/" "demo should check the configured frontend port"
	assert_contains "$temp_dir/demo.out" "UI:      http://override-site.example" "demo should print the effective UI URL"
	assert_contains "$temp_dir/demo.out" "API:     http://override-api.example" "demo should print the effective API URL"
	assert_line_before "$log_file" "just dev up" "curl -fsS http://127.0.0.1:18080/healthz" "demo should start the stack before waiting for the API"

	: >"$log_file"
	PATH="$temp_dir/bin:$PATH" run_just_with_fake_podman "$temp_dir" demo >"$temp_dir/demo.out"
	assert_contains "$log_file" "demo-env API=http://file-api.example SITE=http://file-site.example CORS=http://file-site.example" "demo should use .env public settings when no explicit overrides are set"
	assert_contains "$temp_dir/demo.out" "UI:      http://file-site.example" "demo should print the .env site URL"

	: >"$log_file"
	PATH="$temp_dir/bin:$PATH" run_just_with_fake_podman "$temp_dir" setup
	assert_contains "$log_file" "podman network inspect file_net" "setup should use the network from .env"
	assert_contains "$log_file" "podman network create --opt isolate=strict from_file_jobs" "setup should create the isolated job network for the project from .env"

}

run_diagnose_guidance_case() {
	local temp_dir="$1"
	local output="$temp_dir/diagnose.out"
	local mock_dir="$temp_dir/diagnose-bin"
	mkdir -p "$mock_dir"
	cat >"$mock_dir/go" <<'EOF'
#!/usr/bin/env bash
printf 'go version go1.26.10 linux/amd64\n'
EOF
	cat >"$mock_dir/bun" <<'EOF'
#!/usr/bin/env bash
printf '1.3.8\n'
EOF
	cat >"$mock_dir/node" <<'EOF'
#!/usr/bin/env bash
printf 'v24.0.0\n'
EOF
	cat >"$mock_dir/ss" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
	chmod +x "$mock_dir/"*
	# A socket inode satisfies diagnosis without starting a service.
	python3 - "$temp_dir/podman.sock" <<'PYTHON'
import socket
import sys
with socket.socket(socket.AF_UNIX) as sock:
    sock.bind(sys.argv[1])
PYTHON
	PATH="$mock_dir:$PATH" GO="$mock_dir/go" BUN="$mock_dir/bun" NODE="$mock_dir/node" \
	PODMAN="$temp_dir/podman" PODMAN_SOCKET="$temp_dir/podman.sock" \
	JUST="$REAL_JUST" CURL=true ENV_FILE="$temp_dir/work/.env" \
	STAGEFLOW_TEST_LOG="$temp_dir/podman.log" \
	bash "$REPO_ROOT/infra/scripts/diagnose-local-env.sh" >"$output"
	assert_contains "$output" "Manual bootstrap:" "diagnose should print manual setup guidance"
	assert_line_before "$output" "  just setup" "  just images" "diagnose should install dependencies before building images"
	assert_line_before "$output" "  just images" "  just dev up          # settings come from .env" "diagnose should build images before starting the stack"
	assert_contains "$output" "For localhost scans, set PLATFORM_API_ALLOW_PRIVATE_TARGETS=true, POD_NETNS_MODE=host, and POD_NETWORK= in .env." "diagnose should explain all localhost settings"
	if grep -Fq 'dev init' "$output"; then
		echo "FAIL: diagnose recommends obsolete dev init" >&2
		failures=$((failures + 1))
	fi
}

run_protected_host_case() {
	local temp_dir="$1"
	local current_host
	current_host="$(hostname -f 2>/dev/null || hostname)"
	local recipe
	for recipe in demo dev dev-refresh; do
		: >"$temp_dir/podman.log"
		if STAGEFLOW_PROTECTED_HOST="$current_host" STAGEFLOW_ALLOW_LOCAL_STACKS=0 \
			run_just_with_fake_podman "$temp_dir" "$recipe" >"$temp_dir/guard.log" 2>&1; then
			echo "FAIL: $recipe bypassed the protected-host guard" >&2
			failures=$((failures + 1))
		fi
		assert_contains "$temp_dir/guard.log" "on the protected host" "$recipe must explain the refusal"
		if [[ -s "$temp_dir/podman.log" ]]; then
			echo "FAIL: $recipe invoked Podman on the protected host" >&2
			failures=$((failures + 1))
		fi
	done
	: >"$temp_dir/podman.log"
	STAGEFLOW_PROTECTED_HOST="$current_host" STAGEFLOW_ALLOW_LOCAL_STACKS=1 \
		run_just_with_fake_podman "$temp_dir" dev down
	assert_contains "$temp_dir/podman.log" "podman compose" "the explicit override must allow local stack commands"
}

main() {
	# Just exports the invoking checkout's .env; fixtures must not inherit it.
	unset FRONTEND_PORT PLATFORM_API_PORT VITE_API_URL VITE_SITE_URL STAGEFLOW_DATA_VOLUME
	unset ENV_FILE COMPOSE_PROJECT_NAME STAGEFLOW_NETWORK_NAME STAGEFLOW_JOBS_NETWORK_NAME POD_NETWORK IMAGE_TAG \
		PLATFORM_API_CORS_ALLOW_ORIGINS STAGEFLOW_PUBLIC_DOMAIN STAGEFLOW_PROTECTED_HOST STAGEFLOW_ALLOW_LOCAL_STACKS \
		STAGEFLOW_DEMO_VITE_API_URL STAGEFLOW_DEMO_VITE_SITE_URL \
		STAGEFLOW_DEMO_CORS_ALLOW_ORIGINS STAGEFLOW_DEMO_PUBLIC_DOMAIN

	local temp_dir
	temp_dir="$(mktemp -d)"
	trap 'rm -rf "${temp_dir:-}"' EXIT

	mkdir -p "$temp_dir/work"
	cp "$REPO_ROOT/justfile" "$temp_dir/work/justfile"
	ln -s "$REPO_ROOT/infra" "$temp_dir/work/infra"
	write_fake_podman "${temp_dir}/podman"

	run_protected_host_case "$temp_dir"
	run_network_default_case "$temp_dir"
	run_network_project_case "$temp_dir"
	run_network_override_case "$temp_dir"
	run_env_settings_case "$temp_dir"
	run_external_env_case "$temp_dir"
	run_demo_settings_case "$temp_dir"
	run_diagnose_guidance_case "$temp_dir"

	if [[ "$failures" -ne 0 ]]; then
		echo "dev onboarding tests failed: ${failures}" >&2
		exit 1
	fi

	echo "dev onboarding tests passed"
}

main "$@"
