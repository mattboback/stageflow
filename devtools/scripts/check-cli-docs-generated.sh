#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
docs_dir="$root_dir/docs/reference/cli/stageflow"
tmp_dir="$(mktemp -d)"

cleanup() {
	rm -rf "$tmp_dir"
}
trap cleanup EXIT

generated_dir="$tmp_dir/stageflow.generated"

(cd "$root_dir" && go run ./clients/cli docs --out-dir "$generated_dir")

if ! diff -ru "$docs_dir" "$generated_dir"; then
	echo "CLI docs are stale. Run: go run ./clients/cli docs --out-dir docs/reference/cli/stageflow" >&2
	exit 1
fi
