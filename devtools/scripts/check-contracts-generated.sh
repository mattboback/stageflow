#!/usr/bin/env bash
# Fails when the committed contract code under libs/contracts differs from what
# generate-contracts.sh produces from the schemas.
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"

./devtools/scripts/generate-contracts.sh

# Compare against the index, so staged but uncommitted output also counts as
# current; new files the generator wrote are drift too.
untracked="$(git ls-files --others --exclude-standard -- libs/contracts)"
if ! git diff --quiet -- libs/contracts || [[ -n "$untracked" ]]; then
	git --no-pager diff --stat -- libs/contracts >&2
	[[ -z "$untracked" ]] || echo "untracked: ${untracked//$'\n'/ }" >&2
	echo "Generated contracts are stale. Run: just generate-contracts, then commit the result." >&2
	exit 1
fi
