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
	echo "[go] fmt ${module}"
	(
		cd "${REPO_ROOT}/${module}"
		GOWORK=off golangci-lint fmt -c "${REPO_ROOT}/.golangci.yml"
	)
done < <(go_modules)

if command -v bun >/dev/null 2>&1; then
	echo "[node] format portfolio-frontend"
	if [[ ! -d "${REPO_ROOT}/portfolio/frontend/node_modules" ]]; then
		(cd "${REPO_ROOT}/portfolio/frontend" && bun install --frozen-lockfile)
	fi
	(cd "${REPO_ROOT}/portfolio/frontend" && bun run format)

	echo "[node] format scanner-runner"
	if [[ ! -d "${REPO_ROOT}/platform/scanner-runner/node_modules" ]]; then
		(cd "${REPO_ROOT}/platform/scanner-runner" && bun install --frozen-lockfile)
	fi
	(cd "${REPO_ROOT}/platform/scanner-runner" && bun run format)
else
	echo "[node] bun not found; skipping formatting"
fi
