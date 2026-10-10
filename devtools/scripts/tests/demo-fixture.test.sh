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

# Exercise later guard failures in a copied repository. Even a broken guard
# can only replace this synthetic fixture, never the real public demo.
sandbox="$work/repo"
mkdir -p "$sandbox/devtools/scripts" "$sandbox/qa/e2e" \
  "$sandbox/clients/web/public/demo" "$work/scan-bin" "$work/tmp"
cp "$root/devtools/scripts/build-demo-fixture.sh" "$sandbox/devtools/scripts/"
cp "$root/qa/e2e/scan-helpers.sh" "$sandbox/qa/e2e/"
printf 'preserve this fixture\n' > "$sandbox/clients/web/public/demo/preserve.json"
for tool in dirname mkdir mktemp cp rm sed sort; do
  ln -s "$(type -P "$tool")" "$work/scan-bin/$tool"
done
for tool in stageflow curl magick bun go node; do
  ln -s "$(type -P true)" "$work/scan-bin/$tool"
done
cat > "$work/scan-bin/jq" <<'MOCK'
#!/bin/bash
case "$1" in
  -er) printf 'synthetic-job\n' ;;
  -r) ;;
  *) printf '{}\n' ;;
esac
MOCK
cat > "$work/scan-bin/rg" <<'MOCK'
#!/bin/bash
exit "$DEMO_TEST_RG_STATUS"
MOCK
chmod +x "$work/scan-bin/jq" "$work/scan-bin/rg"
for search_status in 0 2; do
  status=0
  PATH="$work/scan-bin" TMPDIR="$work/tmp" STAGEFLOW_BIN=stageflow \
    DEMO_TEST_RG_STATUS="$search_status" DEMO_ARTIFACT_DIR="$work/artifacts-$search_status" \
    /bin/bash "$sandbox/devtools/scripts/build-demo-fixture.sh" \
    > "$work/guard-$search_status.log" 2>&1 || status=$?
  if [[ "$search_status" == 0 ]]; then
    [[ "$status" == 1 ]]
    grep -Fxq 'Unsafe runtime references survived demo normalization' "$work/guard-$search_status.log"
  else
    [[ "$status" == 2 ]]
    grep -Fxq 'Could not verify demo runtime references (rg exited 2)' "$work/guard-$search_status.log"
  fi
  grep -Fxq 'preserve this fixture' "$sandbox/clients/web/public/demo/preserve.json"
done
echo 'Demo fixture unsafe-reference and search-error guards passed.'
