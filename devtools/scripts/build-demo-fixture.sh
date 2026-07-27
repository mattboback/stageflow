#!/usr/bin/env bash
#
# Regenerate clients/web/public/demo — the static report /demo renders.
#
# StageFlow scans StageFlow. The stack is already the thing being demonstrated,
# so pointing it at the deployed site is both the honest fixture and a standing
# integration test of the whole pipeline. Same dogfooding move as the self-scan
# step in .github/workflows/golden-regression.yml, with two deliberate
# differences:
#
#   - Every scanner runs, not just axe. The gate workflow narrows to axe so a
#     broken outbound link cannot fail an accessibility claim. Here the whole
#     merged report IS the claim, and narrowing would leave the filter chips,
#     the scanner strip and the Lighthouse tab empty on the one page whose job
#     is to show them full.
#
#   - Three URLs, not one. A single-page report has nothing to say about page
#     filtering or the page index.
#
# The result is committed. It has to be: /demo is served by nginx from static
# files with no API behind it, which is the entire point.
#
# Usage:
#   just demo-fixture                 # against an already-running local stack
#   BASE=http://frontend-react:3020 PROBE=http://localhost:3020 just demo-fixture
#
# Requires a running stack (`just demo`), the stageflow CLI on PATH
# (`just cli-install`), plus jq and either cwebp or magick.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
out_dir="${repo_root}/clients/web/public/demo"

# The deployed site, not the local container.
#
# A local container is served over plain http on a private address, and half of
# what the scanners then report is about that rather than about the site:
# "Does not use HTTPS", "Does not redirect HTTP traffic to HTTPS", a robots.txt
# that resolves differently. Rewriting the origin afterwards would leave a
# fixture claiming to be a scan of https://stageflow.org while carrying
# findings that are only true of a container on loopback -- which is precisely
# the fabricated-evidence problem this fixture exists to avoid.
#
# BASE is still overridable for a local target; PROBE is what this script curls
# to check reachability, and differs from BASE only when the scanners resolve
# names the host does not (a pod-networked stack, where the target is a
# service name).
base="${BASE:-https://stageflow.org}"
probe="${PROBE:-$base}"
api="${API:-http://localhost:8080}"
# The URL the committed fixture claims to have scanned. Nobody wants a
# portfolio report whose every page is called localhost.
public_origin="${PUBLIC_ORIGIN:-https://stageflow.org}"
# The scanned page calls its own API, and Lighthouse records the request URL in
# the finding. The built image points VITE_API_URL at the local platform-api,
# so that origin has to be rewritten too or a demo report ships a finding that
# names localhost:8080. Same-origin because that is what the shipped nginx does
# -- it proxies /api/ to platform-api (clients/web/nginx.conf).
api_public_origin="${API_PUBLIC_ORIGIN:-$public_origin}"
# Rewritten into meta.jobId so nothing in the fixture looks like a live job id
# somebody could paste into /scan/<id>/report and get a 404 from.
demo_job_id="demo"
size_limit_bytes=$((2 * 1024 * 1024))

paths=(/ /projects /playground)

die() {
	echo "error: $*" >&2
	exit 1
}

for tool in jq curl stageflow; do
	command -v "$tool" >/dev/null 2>&1 || die "$tool is not on PATH (see the header of this script)"
done

# Either encoder is fine; cwebp is preferred only because its -resize takes a
# zero for "derive this dimension", which is the aspect-ratio guarantee the
# overlay coordinates depend on. ImageMagick's >-suffixed geometry does the
# same thing and is far more likely to be installed already.
if command -v cwebp >/dev/null 2>&1; then
	to_webp() { cwebp -quiet -q 72 -resize 1600 0 "$1" -o "$2"; }
elif command -v magick >/dev/null 2>&1; then
	to_webp() { magick "$1" -resize '1600x>' -quality 72 "$2"; }
else
	die "need cwebp (libwebp-tools) or magick (ImageMagick) to compress screenshots"
fi

curl -fsS "${api}/healthz" >/dev/null 2>&1 || die "no platform-api at ${api} — run 'just demo' first"
curl -fsS "${probe}/" >/dev/null 2>&1 || die "no frontend at ${probe} — run 'just demo' first"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

targets=()
for path in "${paths[@]}"; do
	targets+=("${base}${path}")
done

# Every scanner the deployment has enabled, asked for by name. --scanner
# defaults to four, and the four it picks leave the security-headers,
# open-graph and spelling panels empty on the one page whose job is to show
# them full.
scanners="$(curl -fsS "${api}/api/v1/scanners" |
	jq -r '[.scanners[] | select(.enabled) | .id] | join(",")')"
[[ -n "$scanners" ]] || die "the catalog reports no enabled scanners"

echo "==> Scanning ${#targets[@]} pages with ${scanners} (this takes a few minutes)"
# No --fail-on at all: a demo fixture wants findings, and exiting non-zero on
# the severities it exists to display would be self-defeating.
# --allow-private-targets only when it is actually needed, so a typo in BASE
# cannot silently scan something on the local network.
private_flag=()
case "$base" in
http://localhost* | http://127.* | https://localhost* | http://*:3020*)
	private_flag=(--allow-private-targets)
	;;
esac

stageflow scan "${targets[@]}" \
	--api "$api" \
	--scanner "$scanners" \
	"${private_flag[@]}" \
	--screenshot \
	--format json \
	>"${work}/cli.json" || die "scan failed; see ${work}/cli.json"

# The CLI wraps the report in an envelope: {"schema", "job": {"id"}, "report"}.
job_id="$(jq -r '.job.id // empty' "${work}/cli.json")"
[[ -n "$job_id" ]] || die "could not read a job id out of the CLI output"
jq -e '.report.meta' "${work}/cli.json" >/dev/null || die "CLI output has no .report"

echo "==> Job ${job_id}"

echo "==> Fetching job status for screenshot URLs"
curl -fsS "${api}/api/v1/jobs/${job_id}" >"${work}/status.json" ||
	die "could not fetch job status"

overview_count="$(jq '[.artifacts.screenshots[]? | select(.kind == "page_overview")] | length' "${work}/status.json")"
[[ "$overview_count" -gt 0 ]] ||
	die "the job produced no page_overview screenshots — VisualReviewPanel would render its empty branch"

rm -rf "$out_dir"
mkdir -p "$out_dir"

echo "==> Rewriting the report"
# Four rewrites, each load-bearing:
#   - the scanned origin, so the fixture reads as a scan of the real site
#   - the API origin, which the scanned page calls and Lighthouse records
#   - meta.jobId, so review verdicts key on 'demo' rather than a dead job id
#   - every object-store key (artifacts[].path, scanners[].reportPath and
#     .resultsPath), which are dead references AND carry the real job id that
#     meta.jobId was just rewritten to hide
jq \
	--arg base "$base" \
	--arg origin "$public_origin" \
	--arg api "$api" \
	--arg apiorigin "$api_public_origin" \
	--arg jobid "$demo_job_id" \
	'.report
	 | walk(if type == "string" then gsub($base; $origin) | gsub($api; $apiorigin) else . end)
	 | .meta.jobId = $jobid
	 | if has("artifacts") then .artifacts |= map(del(.path)) else . end
	 | if has("scanners") then .scanners |= map(del(.reportPath, .resultsPath)) else . end' \
	"${work}/cli.json" >"${out_dir}/report.json"

echo "==> Downloading and compressing page overviews"
# VisualReviewPanel draws <image width={pageWidth} height={pageHeight}> inside
# viewBox="0 0 pageWidth pageHeight", so the bitmap is scaled to fit and the
# overlay rectangles stay in page coordinates. Any pixel size works as long as
# the ASPECT RATIO is preserved -- hence -resize 1600 0, which sets the width
# and lets the encoder derive the height. Roughly 900 KB of PNG becomes 120 KB.
screenshots_json="${work}/screenshots.json"
echo '[]' >"$screenshots_json"

while IFS=$'\t' read -r page_id scanner_id page_url url; do
	[[ -n "$page_id" ]] || continue
	curl -fsS "$url" -o "${work}/${page_id}.src" || die "could not download the overview for ${page_id}"
	to_webp "${work}/${page_id}.src" "${out_dir}/${page_id}.webp" ||
		die "webp encoding failed for ${page_id}"

	jq \
		--arg page_id "$page_id" \
		--arg scanner_id "$scanner_id" \
		--arg page_url "${page_url/$base/$public_origin}" \
		--arg file "/demo/${page_id}.webp" \
		'. + [{
			artifact_id: ("demo-" + $page_id),
			scanner_id: $scanner_id,
			page_id: $page_id,
			page_url: $page_url,
			url: $file,
			kind: "page_overview"
		}]' "$screenshots_json" >"${screenshots_json}.next"
	mv "${screenshots_json}.next" "$screenshots_json"
	echo "    ${page_id}  $(du -h "${out_dir}/${page_id}.webp" | cut -f1)"
done < <(jq -r '
	.artifacts.screenshots[]?
	| select(.kind == "page_overview")
	| [.page_id, .scanner_id, (.page_url // ""), .url]
	| @tsv' "${work}/status.json")

cp "$screenshots_json" "${out_dir}/screenshots.json"

echo "==> Checking for anything that should not ship"
leaked="$(grep -lE 'localhost|127\.0\.0\.1|X-Amz-|minio' "${out_dir}"/*.json || true)"
[[ -z "$leaked" ]] || die "presigned URLs or local hostnames survived into: ${leaked}"

total="$(du -sb "$out_dir" | cut -f1)"
echo "==> ${out_dir} is $((total / 1024)) KiB"
[[ "$total" -le "$size_limit_bytes" ]] ||
	die "demo assets exceed $((size_limit_bytes / 1024)) KiB — lower the encoder quality or drop a page"

echo "==> Verifying the fixture against the guard test"
(cd "${repo_root}/clients/web" && bunx vitest run app/lib/demo/demo-fixture.test.ts)

echo "==> Done. Review the diff before committing; this is a portfolio artifact."
