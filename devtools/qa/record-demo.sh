#!/usr/bin/env bash
# Records docs/images/demo.gif: a real CLI scan against the hosted demo API.
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

DEMO_CMD='stageflow scan https://example.com --scanner axe,seo --fail-on serious --max-issues 5 --api https://stageflow.org'

SCRIPT="$(mktemp --suffix=.sh)"
cat >"$SCRIPT" <<EOF
printf '\033[1;36m\$\033[0m '
cmd='$DEMO_CMD'
for ((i = 0; i < \${#cmd}; i++)); do
	printf '%s' "\${cmd:i:1}"
	sleep 0.015
done
printf '\n'
\$cmd
status=\$?
printf '\n\033[1;36m\$\033[0m echo \$?\n%s\n' "\$status"
sleep 2
EOF

asciinema rec --overwrite --cols 100 --rows 32 -c "bash $SCRIPT" "$CAST"
rm -f "$SCRIPT"

agg --idle-time-limit 2 --font-size 15 "$CAST" "$OUT"
ls -lh "$OUT"
