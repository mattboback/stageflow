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
# This script validates fixtures and confirms contract generation still runs.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$REPORT_DIR"

# Always verify fixtures are valid
echo "Validating fixtures..."
if ! bun run validate:fixtures > /dev/null 2>&1; then
    echo "❌ Fixture validation failed. Run: bun run validate:fixtures"
    exit 1
fi
echo "✅ Fixtures valid"

echo "Checking contract generation..."
bun run generate > /dev/null 2>&1
echo "✅ Contract generation works"
exit 0
