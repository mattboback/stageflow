#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# The guard must fail before submitting scans or modifying the public fixture.
mkdir "$work/bin"
ln -s "$(command -v dirname)" "$work/bin/dirname"
for tool in stageflow jq curl magick bun go; do
  ln -s "$(type -P true)" "$work/bin/$tool"
done
status=0
PATH="$work/bin" STAGEFLOW_BIN=stageflow /bin/bash \
  "$root/devtools/scripts/build-demo-fixture.sh" > "$work/output" 2>&1 || status=$?
[[ "$status" == 2 ]]
grep -Fxq 'Missing tool: rg' "$work/output"
echo 'Demo fixture missing-tool guard passed.'
