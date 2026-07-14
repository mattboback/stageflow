#!/usr/bin/env bash
set -euo pipefail

scan_root="${1:-.}"
product_claim_pattern='Docker-based|SARI[F]|Channel not foun[d]'
grep_args=(
    -rn
    --include='*.md' --include='*.yaml' --include='*.yml'
    --include='*.sh' --include='*.go' --include='justfile'
    --include='*.toml' --include='*.json' --include='*.txt'
    --include='*.ts' --include='*.tsx' --include='*.css'
    --exclude-dir='.git' --exclude-dir='node_modules'
    --exclude-dir='vendor' --exclude-dir='.superpowers'
    --exclude-dir='superpowers'
    --exclude='bun.lock' --exclude='go.sum' --exclude='go.work.sum'
    --exclude='check-stale-vocab.sh' --exclude='stale-vocab.test.sh'
)

if [[ ! -d "$scan_root" ]]; then
    echo "FAIL: stale-vocabulary scan root is not a directory: ${scan_root}" >&2
    exit 2
fi

legacy_matches_file="$(mktemp)"
product_matches_file="$(mktemp)"
trap 'rm -f "$legacy_matches_file" "$product_matches_file"' EXIT

scan() {
    local output_file="$1"
    local pattern="$2"
    shift 2
    local status

    set +e
    grep "${grep_args[@]}" "$@" -E "$pattern" -- "$scan_root" >"$output_file"
    status=$?
    set -e

    if [[ "$status" -gt 1 ]]; then
        echo "FAIL: stale-vocabulary scan failed with status ${status}." >&2
        exit "$status"
    fi
}

scan "$legacy_matches_file" '(^|[^a-zA-Z0-9_/.-])(apps/|tools/[a-z]|docs/CONFIGURATION\.md|just run frontend|Scan Worker \(|Release stageflow CLI|project-mode scan using \.stageflow/config\.yaml)'
scan "$product_matches_file" "$product_claim_pattern" \
    --exclude='CHANGELOG.md' --exclude='*.spec.ts' --exclude='*.spec.tsx' \
    --exclude='*.test.ts' --exclude='*.test.tsx'

stale_found=0
while IFS=: read -r file lineno text; do
    [[ -n "$file" ]] || continue
    if [[ "$text" == *golang.org/x/tools* || "$text" == *@apidevtools* ||
        "$text" == *@jsdevtools* || "$text" == *developers.google.com* ||
        "$text" == *detectPackageScriptCommand* || "$text" == *stale-vocab-ok* ||
        "$text" =~ -E.*\(apps/\|tools/ ]]; then
        continue
    fi
    echo "  STALE: ${file}:${lineno}: ${text}" >&2
    stale_found=1
done <"$legacy_matches_file"

while IFS=: read -r file lineno text; do
    [[ -n "$file" ]] || continue
    echo "  STALE: ${file}:${lineno}: ${text}" >&2
    stale_found=1
done <"$product_matches_file"

if [[ "$stale_found" -eq 1 ]]; then
    echo "FAIL: stale vocabulary found. Fix the references above." >&2
    exit 1
fi

echo "  No stale vocabulary found."
