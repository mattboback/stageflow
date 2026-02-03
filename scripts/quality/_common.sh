#!/usr/bin/env bash
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

export XDG_CACHE_HOME="${XDG_CACHE_HOME:-$REPO_ROOT/.cache}"
mkdir -p "$XDG_CACHE_HOME"

export GOCACHE="${GOCACHE:-$XDG_CACHE_HOME/go-build}"
export GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-$XDG_CACHE_HOME/golangci-lint}"

if [[ "${STAGEFLOW_LOCAL_GOMODCACHE:-0}" == "1" ]]; then
	export GOMODCACHE="${GOMODCACHE:-$XDG_CACHE_HOME/go/pkg/mod}"
fi

go_modules() {
	awk '/^[[:space:]]+\.\//{gsub(/^[[:space:]]+/, ""); sub(/^\.\//, ""); print}' "$REPO_ROOT/go.work"
}
