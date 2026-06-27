#!/usr/bin/env bash
#
# Pre-commit check for contracts/provenance schema generation.
#
# Validates fixtures and confirms contract generation still runs.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROVENANCE_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROVENANCE_DIR"

echo "Validating fixtures..."
if ! bun run validate:fixtures > /dev/null 2>&1; then
    echo "Fixture validation failed. Run: bun run validate:fixtures"
    exit 1
fi
echo "Fixtures valid"

echo "Checking contract generation..."
bun run generate > /dev/null 2>&1
echo "Contract generation works"
exit 0
