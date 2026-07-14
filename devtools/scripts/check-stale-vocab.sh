#!/usr/bin/env bash
set -euo pipefail

scan_root="${1:-.}"
product_claim_pattern='Docker-based, rootless container[s]|Artifacts you own: HTML, JSON, SARI[F]|Channel not foun[d]'
grep_args=(
    -rn
    --include='*.md' --include='*.yaml' --include='*.yml'
    --include='*.sh' --include='*.go' --include='justfile'
    --include='*.toml' --include='*.json'
    --include='*.ts' --include='*.tsx' --include='*.css'
    --exclude-dir='.git' --exclude-dir='node_modules'
    --exclude-dir='vendor' --exclude-dir='.superpowers'
    --exclude-dir='superpowers'
)

stale_found=0
while IFS=: read -r file lineno text; do
    echo "  STALE: ${file}:${lineno}: ${text}" >&2
    stale_found=1
done < <(
    {
        grep "${grep_args[@]}" \
            -E '(^|[^a-zA-Z0-9_/.-])(apps/|tools/[a-z]|docs/CONFIGURATION\.md|just run frontend|Scan Worker \(|Release stageflow CLI|project-mode scan using \.stageflow/config\.yaml)' \
            "$scan_root" \
        || true
        grep "${grep_args[@]}" \
            --exclude='CHANGELOG.md' --exclude='*.spec.ts' --exclude='*.spec.tsx' \
            --exclude='*.test.ts' --exclude='*.test.tsx' \
            -E "$product_claim_pattern" \
            "$scan_root" \
        || true
    } \
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
