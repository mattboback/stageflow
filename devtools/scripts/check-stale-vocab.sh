#!/usr/bin/env bash
set -euo pipefail

stale_found=0
while IFS=: read -r file lineno text; do
    echo "  STALE: ${file}:${lineno}: ${text}" >&2
    stale_found=1
done < <(
    grep -rn \
        --include='*.md' --include='*.yaml' --include='*.yml' \
        --include='*.sh' --include='*.go' --include='justfile' \
        --include='*.toml' --include='*.json' \
        --exclude-dir='.git' --exclude-dir='node_modules' \
        --exclude-dir='vendor' --exclude-dir='.storybook' \
        -E '(^|[^a-zA-Z0-9_/.-])(apps/|tools/[a-z]|docs/CONFIGURATION\.md|just run frontend|Scan Worker \(|Release stageflow CLI|project-mode scan using \.stageflow/config\.yaml)' \
        . \
    | grep -v 'bun\.lock' \
    | grep -v 'golang\.org/x/tools' \
    | grep -v 'go\.sum' \
    | grep -v 'go\.work\.sum' \
    | grep -v 'node_modules/' \
    | grep -v '@apidevtools' \
    | grep -v '@jsdevtools' \
    | grep -v 'developers\.google\.com' \
    | grep -v 'detectPackageScriptCommand' \
    | grep -v 'stale-vocab-ok' \
    | grep -v -- '-E .*(apps/|tools/' \
    || true
)

if [[ "$stale_found" -eq 1 ]]; then
    echo "FAIL: stale vocabulary found. Fix the references above." >&2
    exit 1
fi

echo "  No stale vocabulary found."
