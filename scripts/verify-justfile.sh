#!/usr/bin/env bash
# Verify justfile is working correctly

set -euo pipefail

echo "=== Justfile Verification ==="
echo ""

# Check just is installed
if ! command -v just >/dev/null 2>&1; then
    echo "✗ just is not installed"
    echo "  Install with: cargo install just"
    echo "  Or see: https://just.systems"
    exit 1
fi

echo "✓ just is installed ($(just --version))"
echo ""

# Count recipes
RECIPE_COUNT=$(just --summary | wc -w)
echo "✓ Found $RECIPE_COUNT recipes"
echo ""

# Test key commands (dry run to avoid side effects)
echo "Testing key commands (dry run)..."
echo ""

COMMANDS=(
    "help"
    "setup"
    "dev"
    "ci"
    "build"
    "images"
    "prod"
    "deploy"
    "run"
    "clean"
)

PASSED=0
FAILED=0

for cmd in "${COMMANDS[@]}"; do
    case "$cmd" in
        run)
            dry_run_args=(run frontend)
            ;;
        *)
            dry_run_args=("$cmd")
            ;;
    esac

    if just --dry-run "${dry_run_args[@]}" >/dev/null 2>&1; then
        echo "  ✓ just ${dry_run_args[*]}"
        PASSED=$((PASSED + 1))
    else
        echo "  ✗ just ${dry_run_args[*]}"
        FAILED=$((FAILED + 1))
    fi
done

echo ""
echo "Results: $PASSED passed, $FAILED failed"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "✓ All tests passed!"
    echo ""
    echo "Quick start:"
    echo "  just --list          # List all commands"
    echo "  just help            # Show help"
    echo "  just setup           # One-time setup"
    echo "  just dev             # Start local dev stack"
    echo "  just ci              # Lint + typecheck + test"
    echo "  just build           # Build all artifacts"
    echo ""
    exit 0
else
    echo "✗ Some tests failed"
    exit 1
fi
