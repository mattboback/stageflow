#!/usr/bin/env bash
#
# Pre-commit check for contracts/provenance schema generation.
#
# Validates fixtures and verifies that the generated TypeScript and Go are in
# sync with the schema. Mirrors libs/contracts/report/scripts/pre-commit-check.sh.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROVENANCE_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROVENANCE_DIR"

SCHEMA_FILE="schema/provenance.schema.json"
GEN_TS="generated/typescript/provenance.ts"
GEN_GO="generated/go/provenance_schema.go"

if git rev-parse --git-dir > /dev/null 2>&1; then
    STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)

    if echo "$STAGED_FILES" | grep -q "$SCHEMA_FILE"; then
        if ! echo "$STAGED_FILES" | grep -q "$GEN_TS"; then
            echo "Schema modified but TypeScript types not staged."
            echo "  Run: bun run generate:ts && git add $GEN_TS"
        fi
        if ! echo "$STAGED_FILES" | grep -q "$GEN_GO"; then
            echo "Schema modified but Go types not staged."
            echo "  Run: bun run generate:go && git add $GEN_GO"
        fi
    fi
fi

echo "Validating fixtures..."
if ! bun run validate:fixtures > /dev/null 2>&1; then
    echo "Fixture validation failed. Run: bun run validate:fixtures"
    exit 1
fi
echo "Fixtures valid"

echo "Checking generated code freshness..."

TMP_DIR="$(mktemp -d)"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

TS_SNAPSHOT="$TMP_DIR/ts-check.ts"
GO_SNAPSHOT="$TMP_DIR/go-check.go"

cp "$GEN_TS" "$TS_SNAPSHOT"
cp "$GEN_GO" "$GO_SNAPSHOT"

bun run generate > /dev/null 2>&1

TS_DIFF=0
GO_DIFF=0

if ! diff -q "$GEN_TS" "$TS_SNAPSHOT" > /dev/null 2>&1; then
    TS_DIFF=1
fi

if ! diff -q "$GEN_GO" "$GO_SNAPSHOT" > /dev/null 2>&1; then
    GO_DIFF=1
fi

if [[ $TS_DIFF -eq 1 ]]; then
    cp "$TS_SNAPSHOT" "$GEN_TS"
    echo "TypeScript types are out of date."
    echo "  Run: bun run generate:ts"
fi

if [[ $GO_DIFF -eq 1 ]]; then
    cp "$GO_SNAPSHOT" "$GEN_GO"
    echo "Go types are out of date."
    echo "  Run: bun run generate:go"
fi

if [[ $TS_DIFF -eq 1 ]] || [[ $GO_DIFF -eq 1 ]]; then
    exit 1
fi

echo "Generated code is fresh"
exit 0
