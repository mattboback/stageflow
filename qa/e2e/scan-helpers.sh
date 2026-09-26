#!/usr/bin/env bash
# Shared lifecycle for the file-diff golden and deployment smoke checks.
# Callers set REPO_ROOT, API_URL and ARTIFACT_DIR before sourcing this file.

sf() {
  "${STAGEFLOW_BIN:-stageflow}" --api "${API_URL}" "$@"
}

delete_scan() {
  local job_id="$1" response remaining request_timeout
  local deadline=$(( $(date +%s) + 300 ))
  while true; do
    remaining=$(( deadline - $(date +%s) ))
    if (( remaining <= 0 )); then
      echo "FAIL: timed out deleting job ${job_id}; it is still running" >&2
      return 1
    fi
    request_timeout=$(( remaining < 30 ? remaining : 30 ))
    if ! response=$(curl --silent --show-error --max-time "$request_timeout" \
      --output "${WORK_DIR}/delete-${job_id}.txt" --write-out '%{http_code}' \
      "${CURL_AUTH[@]}" --request DELETE "${API_URL}/api/v1/jobs/${job_id}"); then
      return 1
    fi
    case "$response" in
      200|202|204|404)
        echo "Deleted job ${job_id} (HTTP ${response})."
        return 0
        ;;
      409)
        # Running scans cannot be deleted. Give them time to become terminal.
        remaining=$(( deadline - $(date +%s) ))
        if (( remaining > 0 )); then
          sleep "$(( remaining < 2 ? remaining : 2 ))"
        fi
        ;;
      *)
        echo "FAIL: could not delete job ${job_id} (HTTP ${response})" >&2
        return 1
        ;;
    esac
  done
}

cleanup_scans() {
  local status=$? job_id cleanup_failed=0 file
  trap - EXIT
  # A second interrupt must not bypass diagnostics or replace an earlier failure.
  trap 'if [[ "$status" -eq 0 ]]; then status=130; fi' INT
  trap 'if [[ "$status" -eq 0 ]]; then status=143; fi' TERM
  for file in "${WORK_DIR}"/*-stderr.txt; do
    [[ -f "$file" ]] || continue
    # The CLI prints this immediately after intake, including when waiting fails.
    while IFS= read -r job_id; do
      [[ -n "$job_id" ]] || continue
      if ! delete_scan "$job_id"; then
        cleanup_failed=1
      fi
    done < <(sed -nE 's/^Job submitted: ([A-Za-z0-9-]+)$/\1/p' "$file" | sort -u)
  done
  if [[ "$cleanup_failed" -ne 0 && "$status" -eq 0 ]]; then
    status=1
  fi
  if [[ -n "${ARTIFACT_DIR}" ]]; then
    mkdir -p "${ARTIFACT_DIR}"
    cp -a "${WORK_DIR}/." "${ARTIFACT_DIR}/"
  fi
  if [[ "$status" -ne 0 ]]; then
    echo "FAIL: scan check exited ${status}; diagnostics: ${ARTIFACT_DIR:-${WORK_DIR}}" >&2
    # Keep failed runs available for diagnosis if no artifact directory was set.
    [[ -z "${ARTIFACT_DIR}" ]] || rm -rf "${WORK_DIR}"
  else
    rm -rf "${WORK_DIR}"
  fi
  exit "$status"
}

prepare_scans() {
  local cmd
  for cmd in "${STAGEFLOW_BIN:-stageflow}" jq curl node; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      echo "FATAL: ${cmd} not found" >&2
      exit 2
    fi
  done
  API_URL="${API_URL%/}"
  CURL_AUTH=()
  if [[ -n "${STAGEFLOW_API_KEY:-}" ]]; then
    CURL_AUTH=(-H "X-Api-Key: ${STAGEFLOW_API_KEY}")
  fi
  WORK_DIR=$(mktemp -d)
  trap cleanup_scans EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  export NODE_PATH="${REPO_ROOT}/services/scanner-runner/node_modules${NODE_PATH:+:${NODE_PATH}}"
  if ! node -e 'require("ajv"); require("ajv-formats")' >/dev/null 2>&1; then
    echo "FATAL: report validation requires scanner dependencies; run (cd services/scanner-runner && bun install --frozen-lockfile)" >&2
    exit 2
  fi
}

scan_report() {
  local name="$1" target="$2"
  echo "==> Scanning ${name}: ${target}"
  if ! sf scan "$target" --scanner axe --format json --max-issues 0 --timeout 5m \
    >"${WORK_DIR}/${name}-raw.json" 2>"${WORK_DIR}/${name}-stderr.txt"; then
    cat "${WORK_DIR}/${name}-stderr.txt" >&2
    return 1
  fi
  jq -e '.schema == "stageflow-cli/report@v1"' "${WORK_DIR}/${name}-raw.json" >/dev/null
  jq '.report' "${WORK_DIR}/${name}-raw.json" >"${WORK_DIR}/${name}-report.json"
  node "${REPO_ROOT}/libs/contracts/report/schema/validate.js" "${WORK_DIR}/${name}-report.json"
  jq -e '
    .job.state == "DONE" and .job.id == .report.meta.jobId
    and .filters.truncated == false
    and .report.summary.totalIssues == (.report.issues | length)
    and .report.summary.pagesScanned == (.report.pages | length)
    and (.report.pages | length) > 0
    and (.report.scanners | length) == 1
    and .report.scanners[0].id == "axe"
    and .report.scanners[0].status == "success"
    and ((.report.errors // []) | length) == 0
  ' "${WORK_DIR}/${name}-raw.json" >/dev/null
}
