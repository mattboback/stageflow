#!/usr/bin/env bash
set -euo pipefail

# Integration check for the public-edge credential boundary. Caddy and a tiny
# capture upstream share one network namespace so the production 127.0.0.1
# upstream addresses remain unchanged. Podman uses a pod; Docker uses
# `--network container:...`.

PODMAN=${PODMAN:-podman}
CADDY_IMAGE=${CADDY_IMAGE:-docker.io/library/caddy:2.10-alpine}
PYTHON_IMAGE=${PYTHON_IMAGE:-docker.io/library/python:3.13-alpine}
PORT=${STAGEFLOW_CADDY_TEST_PORT:-18443}
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
pod="stageflow-caddy-capture-$$"
capture="${pod}-upstream"
edge="${pod}-edge"
runtime_kind="$(basename "$PODMAN")"

cleanup() {
	if [[ "$runtime_kind" == docker ]]; then
		"$PODMAN" rm -f "$edge" "$capture" >/dev/null 2>&1 || true
	else
		"$PODMAN" pod rm -f "$pod" >/dev/null 2>&1 || true
	fi
}
trap cleanup EXIT

capture_runtime_args=()
edge_runtime_args=()
if [[ "$runtime_kind" == docker ]]; then
	capture_runtime_args=(-p "${PORT}:443")
	edge_runtime_args=(--network "container:${capture}")
else
	"$PODMAN" pod create --name "$pod" -p "${PORT}:443" >/dev/null
	capture_runtime_args=(--pod "$pod")
	edge_runtime_args=(--pod "$pod")
fi

"$PODMAN" run -d "${capture_runtime_args[@]}" --name "$capture" "$PYTHON_IMAGE" python -c '
import json
from http.server import BaseHTTPRequestHandler, HTTPServer
class Handler(BaseHTTPRequestHandler):
    def reply(self):
        body = json.dumps({
            "method": self.command,
            "path": self.path,
            "api_key": self.headers.get("X-Api-Key", ""),
            "authorization": self.headers.get("Authorization", ""),
        }).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    do_GET = reply
    do_POST = reply
    do_PATCH = reply
    do_DELETE = reply
    do_OPTIONS = reply
    def log_message(self, *_): pass
HTTPServer(("127.0.0.1", 8100), Handler).serve_forever()
' >/dev/null
"$PODMAN" run -d "${edge_runtime_args[@]}" --name "$edge" \
	-e STAGEFLOW_PUBLIC_DOMAIN=localhost \
	-e PLATFORM_API_TOKEN=edge-only-token \
	-v "${ROOT_DIR}/infra/caddy/Caddyfile:/etc/caddy/Caddyfile:ro,z" \
	"$CADDY_IMAGE" caddy run --config /etc/caddy/Caddyfile --adapter caddyfile >/dev/null

ready=false
for _ in $(seq 1 30); do
	probe="$(curl -ksS "https://localhost:${PORT}/api/v1/jobs/test-job" 2>/dev/null || true)"
	if [[ "$probe" == *'"api_key": "edge-only-token"'* ]]; then
		ready=true
		break
	fi
	sleep 1
done
if [[ "$ready" != true ]]; then
	echo "Caddy capture test did not become ready" >&2
	"$PODMAN" logs "$edge" >&2 || true
	exit 1
fi

assert_headers() {
	method=$1
	path=$2
	expected_key=$3
	expected_auth=$4
	shift 4

	body="$(curl -ksS --path-as-is -X "$method" "$@" "https://localhost:${PORT}${path}")"
	python3 - "$body" "$method" "$expected_key" "$expected_auth" <<'PY'
import json, sys
payload = json.loads(sys.argv[1])
assert payload["method"] == sys.argv[2], payload
assert payload["api_key"] == sys.argv[3], payload
assert payload["authorization"] == sys.argv[4], payload
PY
}

caller=(-H 'X-Api-Key: caller-token' -H 'Authorization: Bearer caller-token')

# Exact browser submission and read routes overwrite caller credentials.
assert_headers POST '/api/v1/jobs/urls/anonymous?source=test' edge-only-token '' "${caller[@]}"
assert_headers POST '/api/v1/jobs/urls/browser-auth' edge-only-token '' "${caller[@]}"
assert_headers POST '/api/v1/jobs/zip' edge-only-token '' "${caller[@]}"
assert_headers GET '/api/v1/scanners' edge-only-token '' "${caller[@]}"
assert_headers GET '/api/v1/jobs/job-123' edge-only-token '' "${caller[@]}"
assert_headers GET '/api/v1/jobs/job-123/stream' edge-only-token '' "${caller[@]}"
assert_headers GET '/api/v1/jobs/job-123/report' edge-only-token '' "${caller[@]}"
assert_headers GET '/api/v1/jobs/job-123/results?download=1' edge-only-token '' "${caller[@]}"

# Protected resources, method variants, and suffix aliases preserve only the
# caller's credential; the edge token must never enter these requests.
assert_headers POST '/api/v1/jobs/urls' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers GET '/api/v1/projects' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers GET '/api/v1/projects/demo' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers GET '/api/v1/jobs/job-123/diff' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers GET '/api/v1/jobs/urls/anonymous' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers GET '/api/v1/jobs/zip' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers POST '/api/v1/scanners' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers POST '/api/v1/jobs/job-123/results' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers POST '/api/v1/jobs/urls/anonymous/' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers POST '/api/v1/jobs/urls/browser-auth/extra' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers GET '/api/v1/jobs/job-123/results/' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers POST '/api/v1/jobs/%75rls' caller-token 'Bearer caller-token' "${caller[@]}"
assert_headers GET '/api/v1/jobs/job-123/%64iff' caller-token 'Bearer caller-token' "${caller[@]}"

# The secondary React hostname imports the identical boundary.
assert_headers POST '/api/v1/jobs/urls/anonymous' edge-only-token '' -H 'Host: react.localhost' "${caller[@]}"
assert_headers GET '/api/v1/projects' caller-token 'Bearer caller-token' -H 'Host: react.localhost' "${caller[@]}"

echo "Caddy credential-boundary capture tests passed"
