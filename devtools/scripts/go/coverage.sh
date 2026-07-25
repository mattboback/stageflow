#!/usr/bin/env bash
#
# Measures Go test coverage per go.work module and enforces a regression ratchet.
#
# Two deliberate choices:
#
#   -coverpkg=./...  Without it, `go test -cover` credits coverage only to the
#                    package under test. That reports platform-api's
#                    internal/messaging at 0.0% even though an integration test
#                    exercises it end to end against an embedded NATS server —
#                    publishing numbers that understate the suite and make its
#                    best tests look worthless.
#
#   ratchet, not a   A fixed threshold invites the question "why 60?", and a
#   fixed floor      single global floor is meaningless when real per-module
#                    coverage spans 0% to 100%. This compares each module against
#                    a checked-in baseline and fails only on a drop, so the
#                    question becomes "did this change make it worse?" — the same
#                    question StageFlow's own baselines answer for a frontend.
#
# The comparison itself lives in coverage-ratchet.py so it can be tested without
# a 20-module test run; see devtools/scripts/tests/coverage-ratchet.test.sh.
#
# Usage:
#   coverage.sh              measure, compare against the baseline, write reports
#   coverage.sh --update     rewrite the baseline from this run
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
BASELINE="${ROOT_DIR}/devtools/coverage-baseline.json"
OUT_DIR="${COVERAGE_OUT_DIR:-${ROOT_DIR}/artifacts/coverage}"

# How far a module may drop before this fails. Small but non-zero: coverage moves
# a little when code is added with proportionate tests, and a zero-tolerance gate
# would fail on rounding.
ALLOWED_DROP_POINTS="${COVERAGE_ALLOWED_DROP_POINTS:-1.5}"

update_baseline=0
if [[ "${1:-}" == "--update" ]]; then
	update_baseline=1
fi

mkdir -p "$OUT_DIR"
cd "$ROOT_DIR"

measured_tsv="${OUT_DIR}/measured.tsv"
measured_json="${OUT_DIR}/measured.json"
: >"$measured_tsv"

while IFS= read -r dir; do
	[[ -n "$dir" ]] || continue

	module="${dir#./}"
	safe_name="${module//\//__}"
	profile="${OUT_DIR}/${safe_name}.out"

	# A module with no test files reports no coverage rather than failing.
	if ! find "$dir" -name '*_test.go' -print -quit | grep -q .; then
		printf '  %-52s (no tests)\n' "$module"
		continue
	fi

	printf '  %-52s ' "$module"

	if ! (cd "$dir" && go test -coverpkg=./... -coverprofile="$profile" ./... >"${profile}.log" 2>&1); then
		echo "FAILED"
		echo "--- test output for ${module} ---" >&2
		cat "${profile}.log" >&2
		exit 1
	fi

	percent="$(go tool cover -func="$profile" | awk '/^total:/ {gsub(/%/, "", $3); print $3}')"
	if [[ -z "$percent" ]]; then
		percent="0.0"
	fi

	printf '%s%%\n' "$percent"
	printf '%s\t%s\n' "$module" "$percent" >>"$measured_tsv"

	go tool cover -html="$profile" -o "${OUT_DIR}/${safe_name}.html" 2>/dev/null || true
done < <(bash "${ROOT_DIR}/devtools/scripts/go/go-work-dirs.sh")

python3 "${ROOT_DIR}/devtools/scripts/go/coverage-tsv-to-json.py" "$measured_tsv" "$measured_json"

if [[ "$update_baseline" -eq 1 ]]; then
	cp "$measured_json" "$BASELINE"
	echo ""
	echo "Baseline updated: ${BASELINE#"$ROOT_DIR"/}"
	exit 0
fi

if [[ ! -f "$BASELINE" ]]; then
	echo ""
	echo "No coverage baseline yet. Create one with:" >&2
	echo "  just coverage-update" >&2
	exit 1
fi

exec python3 "${ROOT_DIR}/devtools/scripts/go/coverage-ratchet.py" \
	"$BASELINE" "$measured_json" "$ALLOWED_DROP_POINTS"
