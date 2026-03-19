#!/usr/bin/env bash
#
# Pre-commit check for contracts/report schema generation
#
# Usage:
#   ./scripts/pre-commit-check.sh
#
# To install as git hook:
#   ln -sf ../../libs/contracts/report/scripts/pre-commit-check.sh .git/hooks/pre-commit
#
# This script checks if:
#   1. The schema files have been modified
#   2. The generated code is up to date with the schema

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$REPORT_DIR"

# Check if schema was modified in staged files
SCHEMA_FILE="schema/unified-report.v2.schema.json"
GEN_TS="generated/typescript/unified-report.v2.ts"
GEN_GO="generated/go/report_schema.go"

# If running as pre-commit hook, check staged files
if git rev-parse --git-dir > /dev/null 2>&1; then
    STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)

    # Check if schema is staged but generated files are not
    if echo "$STAGED_FILES" | grep -q "$SCHEMA_FILE"; then
        if ! echo "$STAGED_FILES" | grep -q "$GEN_TS"; then
            echo "⚠️  Schema modified but TypeScript types not staged."
            echo "   Run: bun run generate:ts && git add $GEN_TS"
        fi
        if ! echo "$STAGED_FILES" | grep -q "$GEN_GO"; then
            echo "⚠️  Schema modified but Go types not staged."
            echo "   Run: bun run generate:go && git add $GEN_GO"
        fi
    fi
fi

# Always verify fixtures are valid
echo "Validating fixtures..."
if ! bun run validate:fixtures > /dev/null 2>&1; then
    echo "❌ Fixture validation failed. Run: bun run validate:fixtures"
    exit 1
fi
echo "✅ Fixtures valid"

# Check if generated code is fresh
echo "Checking generated code freshness..."

# Save current state
cp "$GEN_TS" /tmp/ts-check.ts
cp "$GEN_GO" /tmp/go-check.go

# Regenerate (suppress output)
bun run generate > /dev/null 2>&1

# Compare
TS_DIFF=0
GO_DIFF=0

if ! diff -q "$GEN_TS" /tmp/ts-check.ts > /dev/null 2>&1; then
    TS_DIFF=1
fi

if ! diff -q "$GEN_GO" /tmp/go-check.go > /dev/null 2>&1; then
    GO_DIFF=1
fi

# Restore original files if they were different (don't auto-fix)
if [[ $TS_DIFF -eq 1 ]]; then
    cp /tmp/ts-check.ts "$GEN_TS"
    echo "❌ TypeScript types are out of date."
    echo "   Run: bun run generate:ts"
fi

if [[ $GO_DIFF -eq 1 ]]; then
    cp /tmp/go-check.go "$GEN_GO"
    echo "❌ Go types are out of date."
    echo "   Run: bun run generate:go"
fi

# Clean up
rm -f /tmp/ts-check.ts /tmp/go-check.go

if [[ $TS_DIFF -eq 1 ]] || [[ $GO_DIFF -eq 1 ]]; then
    exit 1
fi

echo "✅ Generated code is fresh"
exit 0
