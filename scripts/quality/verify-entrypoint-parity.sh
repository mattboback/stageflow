#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/_common.sh"

cd "$REPO_ROOT"

workflow_file=".github/workflows/ci.yml"
justfile_path="justfile"
canonical_cmd="./scripts/quality/ci.sh"

if ! grep -Fq "run: ${canonical_cmd}" "$workflow_file"; then
	echo "workflow parity check failed: ${workflow_file} must run '${canonical_cmd}'" >&2
	exit 1
fi

if ! awk '
	$0 ~ /^ci:$/ {in_ci=1; next}
	in_ci && /^[^[:space:]]/ {in_ci=0}
	in_ci && $0 ~ /^[[:space:]]+\.\/*scripts\/quality\/ci\.sh[[:space:]]*$/ {found=1}
	END {exit found ? 0 : 1}
' "$justfile_path"; then
	echo "workflow parity check failed: justfile ci recipe must call '${canonical_cmd}'" >&2
	exit 1
fi

echo "quality entrypoint parity check passed"
