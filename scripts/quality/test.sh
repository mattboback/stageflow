#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/quality/_common.sh
source "${SCRIPT_DIR}/_common.sh"

cd "$REPO_ROOT"

echo "[go] go work sync"
go work sync

while read -r module; do
	[[ -z "$module" ]] && continue
	echo "[go] test ${module}"
	(
		cd "${REPO_ROOT}/${module}"
		GOWORK=off go test -race -count=1 "$@" ./...
	)
done < <(go_modules)

if command -v bun >/dev/null 2>&1; then
	echo "[node] test frontend"
	if [[ ! -d "${REPO_ROOT}/frontend/node_modules" ]]; then
		(cd "${REPO_ROOT}/frontend" && bun install --frozen-lockfile)
	fi
	(cd "${REPO_ROOT}/frontend" && bun run test)

	echo "[node] test scanner-runner"
	if [[ ! -d "${REPO_ROOT}/platform/scanner-runner/node_modules" ]]; then
		(cd "${REPO_ROOT}/platform/scanner-runner" && bun install --frozen-lockfile)
	fi
	(cd "${REPO_ROOT}/platform/scanner-runner" && bun run test)
else
	echo "[node] bun not found; skipping tests"
fi

echo "[shell] test verify-justfile"
(cd "${REPO_ROOT}" && bash ./scripts/tests/verify-justfile.test.sh)

echo "[contracts] validate fixtures + generated freshness"
(cd "${REPO_ROOT}/packages/contracts/report" && { [[ -d node_modules ]] || bun install --frozen-lockfile; } && ./scripts/pre-commit-check.sh)

echo "[contracts] typecheck"
(cd "${REPO_ROOT}/packages/contracts/report" && { [[ -d node_modules ]] || bun install --frozen-lockfile; } && bun run typecheck)
