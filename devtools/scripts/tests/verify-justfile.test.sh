#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TARGET_SCRIPT="${REPO_ROOT}/scripts/verify-justfile.sh"

failures=0

assert_contains() {
	local file="$1"
	local expected="$2"
	local message="$3"

	if ! grep -Fq "$expected" "$file"; then
		echo "FAIL: ${message}" >&2
		echo "Expected to find: ${expected}" >&2
		failures=$((failures + 1))
	fi
}

assert_exit_code() {
	local actual="$1"
	local expected="$2"
	local message="$3"

	if [[ "$actual" -ne "$expected" ]]; then
		echo "FAIL: ${message} (expected ${expected}, got ${actual})" >&2
		failures=$((failures + 1))
	fi
}

make_fake_just() {
	local bin_dir="$1"

	cat >"${bin_dir}/just" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail

case "${1-}" in
	--version)
		echo "just 1.0.0"
		exit 0
		;;
	--summary)
		echo "help setup dev ci build images prod deploy run clean"
		exit 0
		;;
	--dry-run)
		shift
		if [[ "${JUST_FAIL_DRY_RUN_CMD:-}" == "${1-}" ]]; then
			exit 1
		fi
		exit 0
		;;
	*)
		echo "unexpected args: $*" >&2
		exit 2
		;;
esac
EOF

	chmod +x "${bin_dir}/just"
}

run_success_case() {
	local temp_dir="$1"
	local out_file="${temp_dir}/success.out"
	local fake_bin="${temp_dir}/bin-success"

	mkdir -p "${fake_bin}"
	make_fake_just "${fake_bin}"

	set +e
	PATH="${fake_bin}:/usr/bin:/bin" /usr/bin/bash "${TARGET_SCRIPT}" >"${out_file}" 2>&1
	local rc=$?
	set -e

	assert_exit_code "${rc}" 0 "verify script should succeed when all dry runs pass"
	assert_contains "${out_file}" "Results: 10 passed, 0 failed" "success case should report all commands passing"
	assert_contains "${out_file}" "✓ just run clients/web" "success case should execute run command with clients/web arg"
}

run_failure_case() {
	local temp_dir="$1"
	local out_file="${temp_dir}/failure.out"
	local fake_bin="${temp_dir}/bin-failure"

	mkdir -p "${fake_bin}"
	make_fake_just "${fake_bin}"

	set +e
	JUST_FAIL_DRY_RUN_CMD="build" PATH="${fake_bin}:/usr/bin:/bin" /usr/bin/bash "${TARGET_SCRIPT}" >"${out_file}" 2>&1
	local rc=$?
	set -e

	assert_exit_code "${rc}" 1 "verify script should fail when a dry-run command fails"
	assert_contains "${out_file}" "✗ just build" "failure case should show the failed command"
	assert_contains "${out_file}" "Some tests failed" "failure case should report failure summary"
}

main() {
	local temp_dir
	temp_dir="$(mktemp -d)"
	trap 'rm -rf "${temp_dir:-}"' EXIT

	run_success_case "${temp_dir}"
	run_failure_case "${temp_dir}"

	if [[ "${failures}" -ne 0 ]]; then
		echo "verify-justfile tests failed: ${failures}" >&2
		exit 1
	fi

	echo "verify-justfile tests passed"
}

main "$@"
