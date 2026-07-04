#!/usr/bin/env bash
# Records docs/images/demo.gif: a real CLI scan against the hosted demo API,
# using --format json so the GIF shows the actual machine-readable output
# (pretty-printed, since the CLI's JSON encoder indents by default).
#
# Requires: stageflow on PATH (just cli-install), asciinema, and agg
# (https://github.com/asciinema/agg). Produces a GIF under 4 MB by compressing
# idle time while the scan streams.
#
# Usage: devtools/qa/record-demo.sh [output.gif]
set -euo pipefail

OUT="${1:-docs/images/demo.gif}"
CAST="$(mktemp --suffix=.cast)"
trap 'rm -f "$CAST"' EXIT

DEMO_CMD='stageflow scan https://example.com --scanner axe --format json --max-issues 1 --api https://stageflow.org'

SCRIPT="$(mktemp --suffix=.sh)"
cat >"$SCRIPT" <<EOF
printf '\033[1;36m\$\033[0m '
cmd='$DEMO_CMD'
for ((i = 0; i < \${#cmd}; i++)); do
	printf '%s' "\${cmd:i:1}"
	sleep 0.01
done
printf '\n'
\$cmd
sleep 2
EOF

asciinema rec --overwrite --cols 100 --rows 45 -c "bash $SCRIPT" "$CAST"
rm -f "$SCRIPT"

# The live demo API takes several real seconds to scan; compress dead air
# aggressively and speed up the whole cast so the GIF stays brisk, but hold
# the final JSON output long enough to actually read.
agg --idle-time-limit 0.5 --speed 1.4 --last-frame-duration 2 --font-size 15 "$CAST" "$OUT"
ls -lh "$OUT"
