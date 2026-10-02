#!/usr/bin/env bash
set -Eeuo pipefail

# Renders the production Quadlet units into a temp dir and checks the job
# network boundary: job pods get their own isolated network, never the stack's.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
INSTALLER="${REPO_ROOT}/infra/quadlets/quadlet-install.sh"

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

assert_absent() {
	local file="$1"
	local unexpected="$2"
	local message="$3"

	if grep -Fq -- "$unexpected" "$file"; then
		echo "FAIL: ${message}" >&2
		echo "Did not expect to find: ${unexpected}" >&2
		failures=$((failures + 1))
	fi
}

render() {
	local temp_dir="$1"
	shift

	env "$@" \
		PATH="${temp_dir}/bin:${PATH}" \
		QUADLET_DIR="${temp_dir}/quadlets" \
		USER_SYSTEMD_DIR="${temp_dir}/systemd" \
		SKIP_STAGEFLOW_TARGET_BACKUP=1 \
		ENV_FILE="${temp_dir}/production.env" \
		IMAGE_TAG=testtag \
		"$INSTALLER" >/dev/null
}

main() {
	unset STAGEFLOW_NETWORK_NAME NETWORK_NAME STAGEFLOW_JOBS_NETWORK_NAME

	local temp_dir
	temp_dir="$(mktemp -d)"
	trap 'rm -rf "${temp_dir:-}"' EXIT

	# The installer reloads systemd; a test must never touch the real one.
	mkdir -p "${temp_dir}/bin" "${temp_dir}/quadlets"
	cat >"${temp_dir}/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
printf 'systemctl %s\n' "$*" >>"${STAGEFLOW_TEST_LOG:?}"
EOF
	chmod +x "${temp_dir}/bin/systemctl"
	: >"${temp_dir}/production.env"
	export STAGEFLOW_TEST_LOG="${temp_dir}/systemctl.log"
	: >"$STAGEFLOW_TEST_LOG"

	local network="${temp_dir}/quadlets/stageflow-jobs.network"
	local orchestrator="${temp_dir}/quadlets/stageflow-orchestrator.container"

	# A rendered unit whose template is gone must be stopped and removed.
	: >"${temp_dir}/quadlets/stageflow-retired.network"

	render "$temp_dir"

	assert_contains "$network" "NetworkName=stageflow_jobs" "the job network should use its default name"
	assert_contains "$network" "Options=isolate=strict" "the job network must be isolated from other Podman networks"
	assert_contains "$orchestrator" "Environment=POD_NETWORK=stageflow_jobs" "job pods should join the job network"
	assert_absent "$orchestrator" "POD_NETWORK=stageflow_net" "job pods must not join the stack network"
	assert_contains "$orchestrator" "Requires=stageflow-postgres.service stageflow-jobs-network.service" "the orchestrator should require the job network unit"
	assert_contains "$orchestrator" "After=network-online.target stageflow-postgres.service stageflow-jobs-network.service" "the orchestrator should start after the job network unit"
	assert_contains "${temp_dir}/quadlets/stageflow.pod" "Network=stageflow_net" "the stack pod should stay on the stack network"
	assert_absent "${temp_dir}/quadlets/stageflow.pod" "stageflow_jobs" "the stack pod must not join the job network"

	if grep -l '@[A-Z_]*@' "${temp_dir}/quadlets"/* >/dev/null 2>&1; then
		echo "FAIL: rendered units still contain template placeholders" >&2
		failures=$((failures + 1))
	fi

	if [[ -e "${temp_dir}/quadlets/stageflow-retired.network" ]]; then
		echo "FAIL: a rendered network unit without a template should be pruned" >&2
		failures=$((failures + 1))
	fi
	assert_contains "$STAGEFLOW_TEST_LOG" "systemctl --user stop stageflow-retired-network.service" "pruning should stop the retired network unit"

	render "$temp_dir" STAGEFLOW_NETWORK_NAME=audit_net STAGEFLOW_JOBS_NETWORK_NAME=audit_jobs

	assert_contains "$network" "NetworkName=audit_jobs" "STAGEFLOW_JOBS_NETWORK_NAME should name the job network"
	assert_contains "$orchestrator" "Environment=POD_NETWORK=audit_jobs" "job pods should follow the job network override"
	assert_contains "${temp_dir}/quadlets/stageflow.pod" "Network=audit_net" "the stack pod should follow the stack network override"

	if [[ "$failures" -ne 0 ]]; then
		echo "quadlet install tests failed: ${failures}" >&2
		exit 1
	fi

	echo "quadlet install tests passed"
}

main "$@"
