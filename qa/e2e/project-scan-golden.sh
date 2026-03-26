#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
FIXTURE_DIR="${REPO_ROOT}/qa/fixtures/project-golden"

# --- Configuration ---
API_URL="${STAGEFLOW_API_URL:-http://localhost:8080}"
API_KEY="${STAGEFLOW_API_KEY:-}"
FIXTURE_BASE_URL="${STAGEFLOW_FIXTURE_BASE_URL:-}"
PROJECT_SLUG="qa-golden-$(date +%s)"

if [[ -z "${FIXTURE_BASE_URL}" ]]; then
  case "${API_URL}" in
    http://localhost:*|http://127.0.0.1:*)
      FIXTURE_BASE_URL="http://localhost:3010"
      ;;
    *)
      FIXTURE_BASE_URL="${API_URL}"
      ;;
  esac
fi

FIXTURE_BASE_URL="${FIXTURE_BASE_URL%/}"
BASELINE_URL="${STAGEFLOW_BASELINE_URL:-${FIXTURE_BASE_URL}/qa/baseline.html}"
REGRESSION_URL="${STAGEFLOW_REGRESSION_URL:-${FIXTURE_BASE_URL}/qa/regression.html}"

# --- Helpers ---
failures=0

fail() {
  echo "FAIL: $1" >&2
  failures=$((failures + 1))
}

sf() {
  local args=(--api "${API_URL}")
  if [[ -n "${API_KEY}" ]]; then
    args+=(--api-key "${API_KEY}")
  fi
  stageflow "${args[@]}" "$@"
}

sf_json() {
  sf --format json "$@"
}

normalize_report() {
  jq '
    .cli = {"version":"NORMALIZED","commit":"NORMALIZED","date":"NORMALIZED"}
    | .job.id = "NORMALIZED"
    | .job.created_at = "2000-01-01T00:00:00Z"
    | .job.updated_at = "2000-01-01T00:00:00Z"
    | .links.job = "NORMALIZED"
    | .links.results = "NORMALIZED"
    | .report.meta.jobId = "NORMALIZED"
    | .report.meta.durationMs = 0
    | .report.meta.scannedAt = "2000-01-01T00:00:00.000Z"
    | .report.meta.startedAt = "2000-01-01T00:00:00.000Z"
    | .report.meta.completedAt = "2000-01-01T00:00:00.000Z"
    | (if .report.scanners then
        .report.scanners |= map(
          .durationMs = 0
          | .startedAt = "2000-01-01T00:00:00.000Z"
          | .completedAt = "2000-01-01T00:00:00.000Z"
          | .reportPath = "NORMALIZED"
          | .resultsPath = "NORMALIZED"
        )
      else . end)
    | (if .report.pages then
        .report.pages |= map(
          .durationMs = 0
          | .startedAt = "2000-01-01T00:00:00.000Z"
          | .finishedAt = "2000-01-01T00:00:00.000Z"
        )
      else . end)
  '
}

normalize_diff() {
  jq '
    .baseline.jobId = "NORMALIZED"
    | .current.jobId = "NORMALIZED"
  '
}

# Split combined stdout into separate JSON documents.
# The CLI outputs report JSON, a blank line, then diff JSON.
split_json_docs() {
  local input="$1"
  local report_file="$2"
  local diff_file="$3"

  python3 -c "
import json, sys

text = open(sys.argv[1]).read()
decoder = json.JSONDecoder()
pos = 0
docs = []
while pos < len(text):
    remaining = text[pos:].lstrip()
    if not remaining:
        break
    pos = len(text) - len(remaining)
    obj, end = decoder.raw_decode(text, pos)
    docs.append(obj)
    pos += end

if len(docs) >= 1:
    with open(sys.argv[2], 'w') as f:
        json.dump(docs[0], f, indent=2)
        f.write('\n')
if len(docs) >= 2:
    with open(sys.argv[3], 'w') as f:
        json.dump(docs[1], f, indent=2)
        f.write('\n')
" "$input" "$report_file" "$diff_file"
}

compare_golden() {
  local actual="$1"
  local golden="$2"
  local label="$3"

  if [[ ! -f "${golden}" ]]; then
    echo "WARN: Golden file missing: ${golden}" >&2
    echo "      Creating from actual output. Review and commit." >&2
    cp "${actual}" "${golden}"
    return
  fi

  if ! diff -u "${golden}" "${actual}" > "${WORK_DIR}/diff-${label}.patch" 2>&1; then
    fail "${label}: output differs from golden file"
    echo "--- Diff (${label}) ---" >&2
    cat "${WORK_DIR}/diff-${label}.patch" >&2
    echo "--- End diff ---" >&2
    echo "To update: cp ${actual} ${golden}" >&2
  fi
}

cleanup_project() {
  sf project delete "${PROJECT_SLUG}" >/dev/null 2>&1 || true
}

# --- Prerequisites ---
echo "==> Checking prerequisites..."

for cmd in stageflow jq python3 curl; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "FATAL: ${cmd} not found on PATH" >&2
    exit 2
  fi
done

if ! curl -sf "${API_URL}/healthz" >/dev/null 2>&1; then
  echo "FATAL: API not reachable at ${API_URL}/healthz" >&2
  echo "Start the repo-local acceptance stack first:" >&2
  echo "  just setup" >&2
  echo "  just images" >&2
  echo "  just dev up local" >&2
  echo "  just dev init local" >&2
  exit 2
fi

for page in "${BASELINE_URL}" "${REGRESSION_URL}"; do
  if ! curl -sf --head "${page}" >/dev/null 2>&1; then
    echo "FATAL: Fixture page not accessible: ${page}" >&2
    echo "The repo-local default expects the local overlay frontend at http://localhost:3010." >&2
    echo "If your fixtures are served somewhere else, set STAGEFLOW_FIXTURE_BASE_URL." >&2
    exit 2
  fi
done

# --- Setup ---
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR:-}"; cleanup_project 2>/dev/null || true' EXIT

echo "==> Work dir: ${WORK_DIR}"
echo "==> Project:  ${PROJECT_SLUG}"
echo "==> API:      ${API_URL}"
echo "==> Fixtures: ${FIXTURE_BASE_URL}"
echo ""

# --- Step 1: Create project ---
echo "==> Step 1: Creating project (axe only)..."
sf_json project create "${PROJECT_SLUG}" \
  --url "${BASELINE_URL}" \
  --scanner axe \
  > "${WORK_DIR}/project-create.json" 2>"${WORK_DIR}/create-stderr.txt"
echo "    Done."

# --- Step 2: Baseline scan ---
echo "==> Step 2: Running baseline scan..."
sf_json scan --project "${PROJECT_SLUG}" --timeout 5m \
  > "${WORK_DIR}/baseline-raw.json" 2>"${WORK_DIR}/baseline-stderr.txt"
echo "    Done."

BASELINE_JOB_ID=$(python3 -c "
import json, sys
decoder = json.JSONDecoder()
text = open(sys.argv[1]).read().lstrip()
obj, _ = decoder.raw_decode(text, 0)
print(obj['job']['id'])
" "${WORK_DIR}/baseline-raw.json")
echo "    Job ID: ${BASELINE_JOB_ID}"

BASELINE_ISSUES=$(python3 -c "
import json, sys
decoder = json.JSONDecoder()
text = open(sys.argv[1]).read().lstrip()
obj, _ = decoder.raw_decode(text, 0)
print(obj['report']['summary']['totalIssues'])
" "${WORK_DIR}/baseline-raw.json")

if [[ "${BASELINE_ISSUES}" != "0" ]]; then
  fail "Baseline scan should have 0 issues, got ${BASELINE_ISSUES}"
fi

# --- Step 3: Promote baseline ---
echo "==> Step 3: Promoting baseline..."
sf project promote "${PROJECT_SLUG}" --job-id "${BASELINE_JOB_ID}" \
  > "${WORK_DIR}/promote.txt" 2>&1
echo "    Done."

# --- Step 4: Update project URL to regression page ---
echo "==> Step 4: Updating project URL to regression page..."
sf_json project update "${PROJECT_SLUG}" --url "${REGRESSION_URL}" \
  > "${WORK_DIR}/project-update.json" 2>"${WORK_DIR}/update-stderr.txt"
echo "    Done."

# --- Step 5: Regression scan ---
echo "==> Step 5: Running regression scan..."
set +e
sf_json scan --project "${PROJECT_SLUG}" --timeout 5m \
  > "${WORK_DIR}/regression-raw.json" 2>"${WORK_DIR}/regression-stderr.txt"
REGRESSION_EXIT=$?
set -e
echo "    Done (exit code: ${REGRESSION_EXIT})."

if [[ "${REGRESSION_EXIT}" -ne 1 ]]; then
  fail "Expected exit code 1 for regression, got ${REGRESSION_EXIT}"
fi

# --- Step 6: Normalize ---
echo "==> Step 6: Normalizing output..."

# Baseline is a single JSON doc (no diff — no baseline was set at scan time)
normalize_report < "${WORK_DIR}/baseline-raw.json" > "${WORK_DIR}/baseline-normalized.json"

# Regression has report + diff
split_json_docs "${WORK_DIR}/regression-raw.json" \
  "${WORK_DIR}/regression-report.json" \
  "${WORK_DIR}/regression-diff.json"

if [[ ! -s "${WORK_DIR}/regression-report.json" ]]; then
  fail "Failed to extract report JSON from regression output"
fi
if [[ ! -s "${WORK_DIR}/regression-diff.json" ]]; then
  fail "Failed to extract diff JSON from regression output"
fi

normalize_report < "${WORK_DIR}/regression-report.json" > "${WORK_DIR}/regression-report-normalized.json"
normalize_diff < "${WORK_DIR}/regression-diff.json" > "${WORK_DIR}/regression-diff-normalized.json"

# --- Step 7: Golden comparison ---
echo "==> Step 7: Comparing against golden files..."

compare_golden "${WORK_DIR}/baseline-normalized.json" \
  "${FIXTURE_DIR}/golden-baseline-report.json" \
  "baseline-report"

compare_golden "${WORK_DIR}/regression-report-normalized.json" \
  "${FIXTURE_DIR}/golden-regression-report.json" \
  "regression-report"

compare_golden "${WORK_DIR}/regression-diff-normalized.json" \
  "${FIXTURE_DIR}/golden-regression-diff.json" \
  "regression-diff"

# --- Step 8: Structural assertions ---
echo "==> Step 8: Structural assertions..."

NEW_ISSUES=$(jq '.delta.newIssues' "${WORK_DIR}/regression-diff.json")
if [[ "${NEW_ISSUES}" != "1" ]]; then
  fail "Expected 1 new issue in diff, got ${NEW_ISSUES}"
fi

NEW_RULE=$(jq -r '.new[0].ruleId' "${WORK_DIR}/regression-diff.json")
if [[ "${NEW_RULE}" != "image-alt" ]]; then
  fail "Expected new issue rule 'image-alt', got '${NEW_RULE}'"
fi

NEW_SEVERITY=$(jq -r '.new[0].severity' "${WORK_DIR}/regression-diff.json")
if [[ "${NEW_SEVERITY}" != "critical" ]]; then
  fail "Expected severity 'critical', got '${NEW_SEVERITY}'"
fi

FIXED_ISSUES=$(jq '.delta.fixedIssues' "${WORK_DIR}/regression-diff.json")
if [[ "${FIXED_ISSUES}" != "0" ]]; then
  fail "Expected 0 fixed issues, got ${FIXED_ISSUES}"
fi

# --- Cleanup ---
echo "==> Cleaning up..."
cleanup_project

# --- Results ---
echo ""
if [[ "${failures}" -ne 0 ]]; then
  echo "FAILED: ${failures} assertion(s) failed" >&2
  exit 1
fi

echo "PASSED: All golden test assertions passed"
