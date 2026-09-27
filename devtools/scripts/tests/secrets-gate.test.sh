#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
GITLEAKS="${GITLEAKS:-gitleaks}"
if ! command -v "$GITLEAKS" >/dev/null 2>&1; then
	echo 'Install the pinned secret scanner: go install github.com/zricethezav/gitleaks/v8@v8.30.1' >&2
	exit 1
fi

# A synthetic token lives only in a throwaway Git branch. This proves the real
# rules and --all history coverage work without putting a canary in our history.
umask 077
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT
git init -q -b main "$temp_dir"
cd "$temp_dir"
git config user.name 'Secret gate test'
git config user.email 'secret-gate@example.invalid'
git -c commit.gpgsign=false commit -q --allow-empty -m 'Clean baseline'

scan() {
	"$GITLEAKS" git --redact --no-banner --log-opts=--all \
		--config "$REPO_ROOT/.gitleaks.toml" \
		--gitleaks-ignore-path "$REPO_ROOT/.gitleaksignore" \
		--report-format json --report-path "$temp_dir/report.json" \
		>"$temp_dir/scan.log" 2>&1
}

scan
git switch -q -c canary
mkdir -p services/orchestrator/internal/adapters/runtime
canary_suffix="$(printf 'stageflow synthetic secret gate canary' | sha256sum | cut -c1-36)"
printf 'token = "ghp_%s"\n' "$canary_suffix" > services/orchestrator/internal/adapters/runtime/client.go
git add services/orchestrator/internal/adapters/runtime/client.go
git -c commit.gpgsign=false commit -q -m 'Synthetic canary'
git switch -q main

set +e
scan
status=$?
set -e
if [[ "$status" -ne 1 ]]; then
	echo "Secret gate failed to reject the synthetic history canary (exit $status)" >&2
	exit 1
fi
python3 - "$temp_dir/report.json" <<'PY'
import json
import sys

findings = json.load(open(sys.argv[1], encoding="utf-8"))
assert any(finding["RuleID"] == "github-pat" for finding in findings), "GitHub canary was not detected"
assert all(finding["Secret"] == "REDACTED" for finding in findings), "Secret report was not redacted"
PY

echo 'Secret history gate detects a synthetic token on another branch.'
