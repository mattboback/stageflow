#!/usr/bin/env bash
# Generate the real sample-site v1 -> v2 demo against an already-running API.
set -Eeuo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
api="${API:-http://localhost:8080}"
cli="${STAGEFLOW_BIN:-stageflow}"
for tool in "$cli" jq curl magick bun go rg; do
  command -v "$tool" >/dev/null || { echo "Missing tool: $tool" >&2; exit 2; }
done
export API_URL="$api"
export ARTIFACT_DIR="${DEMO_ARTIFACT_DIR:-}"
# Shared cleanup also recovers submitted job IDs when the CLI wait fails.
export REPO_ROOT="$repo_root"
# shellcheck source=qa/e2e/scan-helpers.sh
source "$repo_root/qa/e2e/scan-helpers.sh"
prepare_scans
work="$WORK_DIR"
mkdir -p "$work/demo"
auth=()
[[ -z "${STAGEFLOW_API_KEY:-}" ]] || auth=(-H "X-Api-Key: ${STAGEFLOW_API_KEY}")
for version in v1 v2; do
  echo "Scanning sample-site/$version"
  "$cli" --api "$api" scan "$repo_root/qa/fixtures/sample-site/$version" \
    --format json --max-issues 0 --screenshot --timeout 10m \
    >"$work/$version.json" 2>"$work/$version-stderr.txt"
  job="$(jq -er '.job.id' "$work/$version.json")"
  curl -fsS "${auth[@]}" "$api/api/v1/jobs/$job" >"$work/$version-status.json"
  name=baseline
  [[ "$version" == v1 ]] || name=report
  # Normalize only runtime identity/origins/timestamps, preserving findings,
  # selectors and screenshot geometry from the actual scan.
  jq --arg job "$job" --arg id "$( [[ "$version" == v1 ]] && echo demo-baseline || echo demo )" '
    .report | walk(if type == "string" then
      split($job) | join("demo")
      | gsub("http://(127\\.0\\.0\\.1|localhost):[0-9]+"; "https://sample.stageflow.org")
    elif type == "object" then
      with_entries(if (.key | test("^(scannedAt|startedAt|completedAt|finishedAt)$")) then
        .value = "2026-09-30T00:00:00.000Z" else . end)
    else . end)
    | .meta.jobId = $id
    | .artifacts |= map(del(.path))
    | .scanners |= map(del(.reportPath, .resultsPath))
  ' "$work/$version.json" >"$work/demo/$name.json"
done
# One bitmap per page: axe first, then a successful scanner fallback.
printf '[]\n' >"$work/demo/screenshots.json"
while IFS=$'\t' read -r page scanner page_url url; do
  # File links are API paths; MinIO's were absolute presigned URLs.
  [[ "$url" == http* ]] || url="$api$url"
  curl -fsS "${auth[@]}" "$url" -o "$work/$page.src"
  magick "$work/$page.src" -resize '1600x>' -quality 72 "$work/demo/$page.webp"
  jq --arg page "$page" --arg scanner "$scanner" \
    --arg page_url "$page_url" --arg file "/demo/$page.webp" '
    . + [{artifact_id: ("demo-"+$page), scanner_id: $scanner,
      page_id: $page, page_url: ($page_url | gsub("http://(127\\.0\\.0\\.1|localhost):[0-9]+"; "https://sample.stageflow.org")),
      url: $file, kind: "page_overview"}]
  ' "$work/demo/screenshots.json" >"$work/screenshots.next"
  mv "$work/screenshots.next" "$work/demo/screenshots.json"
done < <(jq -r '
  [.artifacts.screenshots[]? | select(.kind == "page_overview")]
  | sort_by(.page_id, (if .scanner_id == "axe" then 0 else 1 end))
  | unique_by(.page_id)[] | [.page_id, .scanner_id, (.page_url // ""), .url] | @tsv
' "$work/v2-status.json")
reference_status=0
rg -l 'localhost|127\.0\.0\.1|X-Amz-|minio' "$work/demo"/*.json || reference_status=$?
case "$reference_status" in
  0) echo 'Unsafe runtime references survived demo normalization' >&2; exit 1 ;;
  1) ;; # rg completed successfully without matches.
  *) echo "Could not verify demo runtime references (rg exited $reference_status)" >&2; exit "$reference_status" ;;
esac
for report in "$work/demo/baseline.json" "$work/demo/report.json"; do
  node "$repo_root/libs/contracts/report/schema/validate.js" "$report"
done
rm -rf "$repo_root/clients/web/public/demo"
cp -a "$work/demo" "$repo_root/clients/web/public/demo"
bun "$repo_root/devtools/scripts/summarize-demo.ts"
(cd "$repo_root/clients/web" && bun x prettier --write app/lib/demo/demo-summary.ts app/lib/demo/demo-crop.ts app/lib/demo/demo-terminal.json)
(cd "$repo_root/clients/web" && bun x vitest run app/lib/demo/demo-fixture.test.ts)
echo 'Generated baseline, current report, screenshots and calculated homepage summary.'
