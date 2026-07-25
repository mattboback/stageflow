#!/usr/bin/env bash
#
# Exercises the Go coverage ratchet's decision logic without running any Go
# tests. A gate that silently waves through regressions is worse than no gate, so
# the failing paths are asserted as carefully as the passing one.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
ratchet="$repo_root/devtools/scripts/go/coverage-ratchet.py"
tsv_to_json="$repo_root/devtools/scripts/go/coverage-tsv-to-json.py"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

write_json() {
	printf '%s\n' "$2" >"$work/$1"
}

# Asserts the ratchet's exit status and that its output mentions each expected
# fragment, so a pass/fail flip cannot hide behind matching text.
expect() {
	local label="$1" expected_status="$2" baseline="$3" measured="$4" allowed="$5"
	shift 5

	local status=0
	"$ratchet" "$work/$baseline" "$work/$measured" "$allowed" >"$work/out" 2>&1 || status=$?

	if [[ "$status" -ne "$expected_status" ]]; then
		echo "FAIL ${label}: expected exit ${expected_status}, got ${status}" >&2
		cat "$work/out" >&2
		exit 1
	fi

	local fragment
	for fragment in "$@"; do
		if ! grep -qF -- "$fragment" "$work/out"; then
			echo "FAIL ${label}: output missing '${fragment}'" >&2
			cat "$work/out" >&2
			exit 1
		fi
	done
}

write_json baseline.json '{"libs/go/config": 91.8, "libs/go/diff": 100.0}'

# Unchanged coverage holds.
write_json same.json '{"libs/go/config": 91.8, "libs/go/diff": 100.0}'
expect "unchanged" 0 baseline.json same.json 1.5 "Coverage holding across 2 modules"

# Improvements hold, and the summary reports the real spread.
write_json better.json '{"libs/go/config": 95.0, "libs/go/diff": 100.0}'
expect "improved" 0 baseline.json better.json 1.5 "lowest 95.0%, highest 100.0%"

# A drop inside the tolerance holds; the same drop plus a hair fails. Asserting
# both sides pins the boundary, which is the only part of the threshold that can
# regress unnoticed.
write_json within.json '{"libs/go/config": 90.3, "libs/go/diff": 100.0}'
expect "drop within tolerance" 0 baseline.json within.json 1.5 "Coverage holding"

write_json beyond.json '{"libs/go/config": 90.2, "libs/go/diff": 100.0}'
expect "drop beyond tolerance" 1 baseline.json beyond.json 1.5 \
	"regressed by more than 1.5 points" \
	"libs/go/config: 91.8% -> 90.2% (-1.6 points)" \
	"just coverage-update"

# Deleting a module's tests reports no coverage rather than 0%, so absence has to
# fail too — otherwise `rm *_test.go` is a green diff.
write_json dropped.json '{"libs/go/diff": 100.0}'
expect "module lost its tests" 1 baseline.json dropped.json 1.5 \
	"libs/go/config: was 91.8%, now reports no coverage"

# A brand-new module is surfaced for recording but does not fail the build.
write_json added.json \
	'{"libs/go/config": 91.8, "libs/go/diff": 100.0, "libs/go/newthing": 42.0}'
expect "new module" 0 baseline.json added.json 1.5 \
	"New modules, not yet in the baseline" \
	"libs/go/newthing: 42.0%"

# A new module alongside a regression still fails on the regression.
write_json added_and_worse.json \
	'{"libs/go/config": 50.0, "libs/go/diff": 100.0, "libs/go/newthing": 42.0}'
expect "new module with a regression" 1 baseline.json added_and_worse.json 1.5 \
	"libs/go/newthing: 42.0%" \
	"libs/go/config: 91.8% -> 50.0%"

# Unreadable inputs fail loudly instead of being treated as "nothing regressed".
write_json malformed.json 'not json at all'
expect "malformed measured file" 1 baseline.json malformed.json 1.5 "coverage ratchet:"

write_json array.json '["not", "an", "object"]'
expect "wrong JSON shape" 1 baseline.json array.json 1.5 \
	"expected a JSON object of module -> percent"

status=0
"$ratchet" >"$work/out" 2>&1 || status=$?
if [[ "$status" -ne 1 ]]; then
	echo "FAIL missing arguments: expected exit 1, got ${status}" >&2
	exit 1
fi

# The TSV shim feeding the ratchet: stable ordering, trailing newline, blank lines
# tolerated.
printf 'libs/go/diff\t100.0\nlibs/go/config\t91.8\n\n' >"$work/measured.tsv"
python3 "$tsv_to_json" "$work/measured.tsv" "$work/measured.json"
expected='{
  "libs/go/config": 91.8,
  "libs/go/diff": 100.0
}'
if [[ "$(cat "$work/measured.json")" != "$expected" ]]; then
	echo "FAIL tsv-to-json produced unexpected output:" >&2
	cat "$work/measured.json" >&2
	exit 1
fi

echo "coverage ratchet tests passed"
