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
	echo "[go] lint ${module}"
	(
		cd "${REPO_ROOT}/${module}"
		GOWORK=off golangci-lint run --timeout=5m -c "${REPO_ROOT}/.golangci.yml"
	)
done < <(go_modules)

if command -v bun >/dev/null 2>&1; then
	echo "[node] lint portfolio-frontend"
	if [[ ! -d "${REPO_ROOT}/portfolio/frontend/node_modules" ]]; then
		(cd "${REPO_ROOT}/portfolio/frontend" && bun install --frozen-lockfile)
	fi
	(cd "${REPO_ROOT}/portfolio/frontend" && bun run lint)
	(cd "${REPO_ROOT}/portfolio/frontend" && bun run type-check)

	echo "[node] lint scanner-runner"
	if [[ ! -d "${REPO_ROOT}/platform/scanner-runner/node_modules" ]]; then
		(cd "${REPO_ROOT}/platform/scanner-runner" && bun install --frozen-lockfile)
	fi
	(cd "${REPO_ROOT}/platform/scanner-runner" && bun run lint)
	(cd "${REPO_ROOT}/platform/scanner-runner" && bun run type-check)
else
	echo "[node] bun not found; skipping lint/typecheck"
fi

if command -v shellcheck >/dev/null 2>&1; then
	echo "[shell] shellcheck"
	git ls-files '*.sh' | xargs -r shellcheck
else
	echo "[shell] shellcheck not installed; skipping"
fi
