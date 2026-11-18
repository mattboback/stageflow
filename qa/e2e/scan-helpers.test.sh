#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ "${1:-}" == --case ]]; then
  case_dir="$2"
  mode="$3"
  initial_status="$4"
  WORK_DIR="$case_dir/work"
  ARTIFACT_DIR="$5"
  API_URL=https://example.invalid
  CURL_AUTH=()
  # shellcheck source=qa/e2e/scan-helpers.sh
  source "$SCRIPT_DIR/scan-helpers.sh"

  # Curl runs in a command substitution, so keep mock state in files.
  # These mocks are invoked by the sourced cleanup helper.
  # shellcheck disable=SC2329
  curl() {
    local attempts response=409 output=''
    attempts=$(<"$case_dir/attempts")
    printf '%s\n' "$((attempts + 1))" >"$case_dir/attempts"
    while [[ $# -gt 0 ]]; do
      if [[ "$1" == --output ]]; then
        output="$2"
      fi
      shift
    done
    if [[ "$mode" == immediate || ( "$mode" != exhausted && "$attempts" -ge 2 ) ]]; then
      response=204
    fi
    printf 'mock response %s\n' "$response" >"$output"
    printf '%s' "$response"
  }
  # shellcheck disable=SC2329
  date() { cat "$case_dir/clock"; }
  # shellcheck disable=SC2329
  sleep() {
    local now
    [[ "$1" == 2 ]] || exit 99
    now=$(<"$case_dir/clock")
    printf '%s\n' "$((now + $1))" >"$case_dir/clock"
    if [[ "$mode" == interrupt && "$now" -eq 0 ]]; then
      kill -INT "$$"
    fi
  }
  trap cleanup_scans EXIT
  exit "$initial_status"
fi

temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT

run_case() {
  local name="$1" mode="$2" initial_status="$3" expected_status="$4" attempts="$5" copy_artifacts="$6"
  local case_dir="$temporary/$name" artifact_dir='' actual_status=0 diagnostics
  mkdir -p "$case_dir/work"
  printf 'Job submitted: job-1\n' >"$case_dir/work/url-stderr.txt"
  printf '0\n' >"$case_dir/attempts"
  printf '0\n' >"$case_dir/clock"
  if [[ "$copy_artifacts" == yes ]]; then
    artifact_dir="$case_dir/artifacts"
  fi
  bash "$SCRIPT_DIR/scan-helpers.test.sh" --case "$case_dir" "$mode" "$initial_status" "$artifact_dir" \
    >"$case_dir/output" 2>&1 || actual_status=$?
  if [[ "$actual_status" != "$expected_status" || "$(<"$case_dir/attempts")" != "$attempts" ]]; then
    cat "$case_dir/output" >&2
    echo "FAIL: $name status=$actual_status attempts=$(<"$case_dir/attempts")" >&2
    exit 1
  fi
  diagnostics="${artifact_dir:-$case_dir/work}"
  if [[ "$expected_status" == 0 ]]; then
    [[ ! -e "$case_dir/work" ]]
  else
    [[ -s "$diagnostics/url-stderr.txt" && -s "$diagnostics/delete-job-1.txt" ]]
    if [[ -n "$artifact_dir" ]]; then
      [[ ! -e "$case_dir/work" ]]
    fi
  fi
  if [[ "$mode" == exhausted ]]; then
    [[ "$(<"$case_dir/clock")" == 300 ]]
  fi
}

run_case immediate immediate 0 0 1 no
run_case running-then-done retry 0 0 3 no
run_case deadline exhausted 0 1 150 no
run_case original-failure exhausted 7 7 150 yes
run_case original-interrupt exhausted 130 130 150 no
run_case original-termination retry 143 143 3 yes
run_case interrupt-during-cleanup interrupt 0 130 3 yes
run_case failure-then-interrupt interrupt 7 7 3 no
echo 'Scan cleanup tests passed'
