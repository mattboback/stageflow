#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

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

run_shadowed_path_case() {
	local temp_dir="$1"
	local out_file="${temp_dir}/shadowed-path.out"
	local fake_bin="${temp_dir}/fake-bin"
	local install_bin="${temp_dir}/install-bin"

	mkdir -p "${fake_bin}" "${install_bin}"
	printf '#!/usr/bin/env bash\necho fake-stageflow\n' >"${fake_bin}/stageflow"
	chmod +x "${fake_bin}/stageflow"

	set +e
	PATH="${fake_bin}:${install_bin}:${PATH}" "${REPO_ROOT}/devtools/scripts/install-cli.sh" "${install_bin}" >"${out_file}" 2>&1
	local rc=$?
	set -e

	assert_exit_code "${rc}" 1 "cli-install should fail when another stageflow shadows the installed binary"
	assert_contains "${out_file}" "does not resolve to the installed binary" "cli-install should explain the PATH shadowing problem"
}

run_bin_dir_missing_from_path_case() {
	local temp_dir="$1"
	local out_file="${temp_dir}/missing-path.out"
	local install_bin="${temp_dir}/off-path-bin"
	local bin_name="stageflow-offpath-test"

	mkdir -p "${install_bin}"

	set +e
	"${REPO_ROOT}/devtools/scripts/install-cli.sh" "${install_bin}" "${bin_name}" >"${out_file}" 2>&1
	local rc=$?
	set -e

	assert_exit_code "${rc}" 1 "cli-install should fail when the install directory is not on PATH"
	assert_contains "${out_file}" "is not on PATH" "cli-install should explain how to fix a missing PATH entry"
}

run_static_install_case() {
	local temp_dir="$1"
	local out_file="${temp_dir}/static-install.out"
	local install_bin="${temp_dir}/static-bin"
	local bin_name="stageflow-static-test"

	mkdir -p "${install_bin}"

	set +e
	CGO_ENABLED=1 PATH="${install_bin}:${PATH}" \
		"${REPO_ROOT}/devtools/scripts/install-cli.sh" "${install_bin}" "${bin_name}" >"${out_file}" 2>&1
	local rc=$?
	set -e

	assert_exit_code "${rc}" 0 "cli-install should install and execute the portable binary"
	if [[ "${rc}" -eq 0 ]]; then
		go version -m "${install_bin}/${bin_name}" >"${temp_dir}/build-info.out"
		assert_contains "${temp_dir}/build-info.out" "CGO_ENABLED=0" \
			"cli-install should disable cgo even when the caller enables it"
	fi
}

main() {
	local temp_dir
	temp_dir="$(mktemp -d)"
	trap 'rm -rf "${temp_dir:-}"' EXIT

	cd "${REPO_ROOT}"
	run_static_install_case "${temp_dir}"
	run_shadowed_path_case "${temp_dir}"
	run_bin_dir_missing_from_path_case "${temp_dir}"

	if [[ "${failures}" -ne 0 ]]; then
		echo "cli-install tests failed: ${failures}" >&2
		exit 1
	fi

	echo "cli-install tests passed"
}

main "$@"
