#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BASE_URL="${1:-${SMOKE_BASE_URL:-}}"

if [[ -z "${BASE_URL}" ]]; then
  if [[ -f "${REPO_ROOT}/.env" ]]; then
    # shellcheck disable=SC1091
    source "${REPO_ROOT}/.env"
  fi
  if [[ -n "${STAGEFLOW_PUBLIC_DOMAIN:-}" ]]; then
    BASE_URL="https://${STAGEFLOW_PUBLIC_DOMAIN}"
  fi
fi

if [[ -z "${BASE_URL}" ]]; then
  echo "smoke: missing base URL. Usage: ./scripts/smoke.sh https://your-domain.com" >&2
  exit 1
fi

if [[ "${BASE_URL}" != http* ]]; then
  BASE_URL="https://${BASE_URL}"
fi

command -v curl >/dev/null 2>&1 || { echo "smoke: missing curl" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "smoke: missing jq" >&2; exit 1; }

"${SCRIPT_DIR}/test-event-flow.sh" "${BASE_URL}"

# Optional public endpoint check: accept 200 or 302 for Grafana (login redirect)
status=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${BASE_URL}/monitoring/") || status="000"
if [[ "${status}" != "200" && "${status}" != "302" ]]; then
  echo "[WARN] Grafana endpoint returned ${status} (expected 200/302)." >&2
fi

printf "\nPublic smoke finished for %s\n" "${BASE_URL}"
