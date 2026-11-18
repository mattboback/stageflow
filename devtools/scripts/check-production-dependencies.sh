#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."
staging=$(mktemp -d)
trap 'rm -rf "$staging"' EXIT

# Preserve the relative path used by the scanner's local contract dependency.
for manifest in libs/contracts/*/package.json; do
    mkdir -p "$staging/$(dirname "$manifest")"
    cp "$manifest" "$staging/$manifest"
done

for workspace in clients/web services/scanner-runner; do
    mkdir -p "$staging/$workspace"
    cp "$workspace/package.json" "$workspace/bun.lock" "$staging/$workspace/"
    (
        cd "$staging/$workspace"
        "${BUN:-bun}" install --production --frozen-lockfile --ignore-scripts
        installed=$(find node_modules -path '*/node_modules/picomatch' -o -path 'node_modules/picomatch')
        if [[ -n "$installed" ]]; then
            echo "$workspace ships picomatch. Review the development-only audit exception." >&2
            exit 1
        fi
    )
done
