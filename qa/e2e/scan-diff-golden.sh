#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
FIXTURE_DIR="${REPO_ROOT}/qa/fixtures/scan-diff-golden"
API_URL="${STAGEFLOW_API_URL:-http://localhost:8080}"
ARTIFACT_DIR="${SCAN_DIFF_GOLDEN_ARTIFACT_DIR:-}"

# shellcheck source=qa/e2e/scan-helpers.sh
source "${SCRIPT_DIR}/scan-helpers.sh"

# Only transport/timing values vary between runs. Keep issue IDs, evidence,
# screenshot geometry, scanner versions, scores and counters in the goldens.
# Screenshot overlays inject a random sf-semantic UUID into contextHtml; only
# that instrumentation ID and the screenshot-only caret style are normalized,
# keeping the surrounding DOM evidence.
normalize_report() {
  jq --sort-keys '
    .job.id as $job
    | walk(if type == "object" then
        with_entries(
          if (.key | test("^(created_at|updated_at|scannedAt|startedAt|completedAt|finishedAt)$")) then
            .value = "2000-01-01T00:00:00.000Z"
          elif .key == "durationMs" then .value = 0
          elif (.key == "reportPath" or .key == "resultsPath") then .value = "NORMALIZED"
          else . end)
      elif type == "string" then
        split($job) | join("NORMALIZED")
        | gsub("sf-semantic-[0-9a-f-]{36}"; "sf-semantic-NORMALIZED")
        | gsub(" style=\"caret-color: transparent !important;\""; "")
        | gsub("http://(127\\.0\\.0\\.1|localhost):[0-9]+"; "https://sample.invalid")
      else . end)
    | .cli = {version: "NORMALIZED", commit: "NORMALIZED", date: "NORMALIZED"}
    | .api.base_url = "https://api.invalid"
    | .links = {job: "https://api.invalid/job", results: "https://api.invalid/results"}
    | .report.artifacts |= map(if has("path") then .path = "https://artifacts.invalid/NORMALIZED" else . end)
  '
}

normalize_diff() {
  jq --sort-keys '
    .baseline.file = "v1.json"
    | .current.file = "v2.json"
    | .current.jobId = "NORMALIZED"
    | walk(if type == "string" then
        gsub("sf-semantic-[0-9a-f-]{36}"; "sf-semantic-NORMALIZED")
        | gsub(" style=\"caret-color: transparent !important;\""; "")
        | gsub("http://(127\\.0\\.0\\.1|localhost):[0-9]+"; "https://sample.invalid")
      else . end)
  '
}

compare_golden() {
  local actual="$1" golden="$2"
  if [[ "${UPDATE_GOLDENS:-0}" == 1 ]]; then
    cp "$actual" "$golden"
    echo "UPDATE: ${golden#"${REPO_ROOT}/"}"
  elif [[ ! -f "$golden" ]]; then
    echo "FAIL: missing ${golden}; regenerate explicitly with UPDATE_GOLDENS=1" >&2
    return 1
  elif ! diff -u "$golden" "$actual"; then
    echo "FAIL: golden changed; inspect the delta before using UPDATE_GOLDENS=1" >&2
    return 1
  fi
}

prepare_scans
scan_report baseline "${REPO_ROOT}/qa/fixtures/sample-site/v1"
scan_report regression "${REPO_ROOT}/qa/fixtures/sample-site/v2"

# Each archive must discover the same four routes. Issue identities are retained
# in the full snapshots below; these assertions also refuse to bless an
# unexpected scanner result when UPDATE_GOLDENS=1 is requested.
for name in baseline regression; do
  jq -e '
    [.report.pages[].path] | sort == ["/about.html", "/index.html", "/pricing.html", "/signup.html"]
  ' "${WORK_DIR}/${name}-raw.json" >/dev/null
done
jq -e '
  [.report.issues[] | [.ruleId, .id]] | sort == [
    ["color-contrast", "74cd9a0218af"],
    ["image-alt", "13ed81627c04"],
    ["label", "1471cd3d6e06"]
  ]
' "${WORK_DIR}/baseline-raw.json" >/dev/null
jq -e '
  [.report.issues[] | [.ruleId, .id]] | sort == [
    ["button-name", "2b9345de2405"],
    ["color-contrast", "74cd9a0218af"],
    ["label", "1471cd3d6e06"],
    ["link-name", "4091e74b22b6"],
    ["select-name", "e3af3e13d504"]
  ]
' "${WORK_DIR}/regression-raw.json" >/dev/null

status=0
sf diff "${WORK_DIR}/baseline-raw.json" "${WORK_DIR}/regression-raw.json" \
  --format json --fail-on-new=any >"${WORK_DIR}/regression-diff.json" || status=$?
if [[ "$status" -ne 1 ]]; then
  echo "FAIL: diff must exit 1 for the planted regression, got ${status}" >&2
  exit 1
fi
jq -e '
  .schema == "stageflow/diff@v1" and .regressed == true
  and .delta.newIssues == 3 and .delta.fixedIssues == 1 and .delta.unchangedIssues == 2
  and ([.new[].ruleId] | sort) == ["button-name", "link-name", "select-name"]
  and [.fixed[].ruleId] == ["image-alt"]
' "${WORK_DIR}/regression-diff.json" >/dev/null

sf diff "${WORK_DIR}/baseline-raw.json" "${WORK_DIR}/baseline-raw.json" \
  --format json --fail-on-new=any >"${WORK_DIR}/unchanged-diff.json"
jq -e '
  .regressed == false and .delta.newIssues == 0
  and .delta.fixedIssues == 0 and .delta.unchangedIssues == 3
' "${WORK_DIR}/unchanged-diff.json" >/dev/null

normalize_report <"${WORK_DIR}/baseline-raw.json" >"${WORK_DIR}/baseline-normalized.json"
normalize_report <"${WORK_DIR}/regression-raw.json" >"${WORK_DIR}/regression-normalized.json"
normalize_diff <"${WORK_DIR}/regression-diff.json" >"${WORK_DIR}/diff-normalized.json"
compare_golden "${WORK_DIR}/baseline-normalized.json" "${FIXTURE_DIR}/golden-baseline-report.json"
compare_golden "${WORK_DIR}/regression-normalized.json" "${FIXTURE_DIR}/golden-regression-report.json"
compare_golden "${WORK_DIR}/diff-normalized.json" "${FIXTURE_DIR}/golden-regression-diff.json"
echo "PASSED: exactly 3 new, 1 fixed and 2 unchanged findings; deleting both jobs."
