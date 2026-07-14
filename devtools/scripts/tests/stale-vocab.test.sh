#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
CHECKER="${REPO_ROOT}/devtools/scripts/check-stale-vocab.sh"
temp_dir=""
trap 'rm -rf "${temp_dir:-}"' EXIT

failures=0

assert_exit_code() {
	local actual="$1"
	local expected="$2"
	local message="$3"

	if [[ "$actual" -ne "$expected" ]]; then
		echo "FAIL: ${message} (expected ${expected}, got ${actual})" >&2
		failures=$((failures + 1))
	fi
}

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

run_checker() {
	local root="$1"
	local output="$2"

	set +e
	"$CHECKER" "$root" >"$output" 2>&1
	local rc=$?
	set -e

	printf '%s' "$rc"
}

run_clean_case() {
	local temp_dir="$1"
	local fixture="${temp_dir}/clean"
	local output="${temp_dir}/clean.out"
	local docker_claim="Docker-based, rootless cont"'ainers'
	local artifact_claim="Artifacts you own: HTML, JSON, SAR"'IF'
	local missing_claim="Channel not "'found'
	local rc

	mkdir -p "$fixture"
	printf '%s\n%s\n%s\n' "$docker_claim" "$artifact_claim" "$missing_claim" >"${fixture}/CHANGELOG.md"
	printf '%s\n' "Current product language" >"${fixture}/README.md"
	rc="$(run_checker "$fixture" "$output")"

	assert_exit_code "$rc" 0 "clean active sources and historical changelog prose should pass"
}

run_stale_case() {
	local temp_dir="$1"
	local name="$2"
	local filename="$3"
	local claim="$4"
	local fixture="${temp_dir}/${name}"
	local output="${temp_dir}/${name}.out"
	local rc

	mkdir -p "$fixture"
	printf '%s\n' "$claim" >"${fixture}/${filename}"
	rc="$(run_checker "$fixture" "$output")"

	assert_exit_code "$rc" 1 "${filename} should fail for stale product language"
	assert_contains "$output" "STALE:" "${filename} should emit a STALE diagnostic"
}

run_invalid_root_case() {
	local output="${temp_dir}/invalid-root.out"
	local rc

	rc="$(run_checker "${temp_dir}/does-not-exist" "$output")"
	assert_exit_code "$rc" 2 "a missing scan root should fail closed"
	assert_contains "$output" "scan root is not a directory" "a missing root should explain the failure"
}

main() {
	temp_dir="$(mktemp -d)"

	run_clean_case "$temp_dir"
	run_stale_case "$temp_dir" docker app.ts "Docker-based, rootless cont"'ainers'
	run_stale_case "$temp_dir" artifacts report.tsx "Artifacts you own: HTML, JSON, SAR"'IF'
	run_stale_case "$temp_dir" missing styles.css "Channel not "'found'
	run_stale_case "$temp_dir" public-copy llms.txt "SARI"'F'
	run_stale_case "$temp_dir" ignored-token README.md "Docker-based developers.google.com"
	run_invalid_root_case

	if [[ "$failures" -ne 0 ]]; then
		echo "stale-vocab tests failed: ${failures}" >&2
		exit 1
	fi

	echo "stale-vocab tests passed"
}

main "$@"
