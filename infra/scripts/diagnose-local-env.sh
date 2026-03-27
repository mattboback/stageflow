#!/usr/bin/env bash
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GO_BIN="${GO:-go}"
BUN_BIN="${BUN:-bun}"
PODMAN_BIN="${PODMAN:-podman}"
JUST_BIN="${JUST:-just}"
CURL_BIN="${CURL:-curl}"
ENV_FILE="${ENV_FILE:-$REPO_ROOT/.env}"
EXPECTED_GO_MIN="1.26.1"
EXPECTED_BUN_MIN="1.3.8"

fatal_count=0
warn_count=0

say() {
	printf '%s\n' "$*"
}

pass() {
	say "  OK   $*"
}

warn() {
	say "  WARN $*"
	warn_count=$((warn_count + 1))
}

fail() {
	say "  FAIL $*"
	fatal_count=$((fatal_count + 1))
}

version_ge() {
	local actual="$1"
	local minimum="$2"

	[[ "$(printf '%s\n%s\n' "$minimum" "$actual" | sort -V | tail -n1)" == "$actual" ]]
}

require_command() {
	local command_name="$1"
	local label="$2"

	if command -v "$command_name" >/dev/null 2>&1; then
		pass "$label available: $(command -v "$command_name")"
		return
	fi

	fail "Missing $label ($command_name)."
}

check_image() {
	local image_name="$1"

	if "$PODMAN_BIN" image exists "$image_name"; then
		pass "Container image present: $image_name"
	else
		warn "Container image missing: $image_name. Run \`just images\` before starting the stack."
	fi
}

read_env_value() {
	local key="$1"
	local line

	line="$(grep -E "^${key}=" "$ENV_FILE" | tail -n1 || true)"
	[[ -n "$line" ]] || return 1
	printf '%s' "${line#*=}"
}

say "==> StageFlow local diagnose"

require_command "$GO_BIN" "Go"
require_command "$BUN_BIN" "Bun"
require_command "$PODMAN_BIN" "Podman"
require_command "$JUST_BIN" "just"
require_command "$CURL_BIN" "curl"

if command -v "$GO_BIN" >/dev/null 2>&1; then
	go_version="$("$GO_BIN" version | awk '{print $3}' | sed 's/^go//')"
	if version_ge "$go_version" "$EXPECTED_GO_MIN"; then
		pass "Go version $go_version (need >= $EXPECTED_GO_MIN)"
	else
		fail "Go version $go_version is too old (need >= $EXPECTED_GO_MIN)."
	fi
fi

if command -v "$BUN_BIN" >/dev/null 2>&1; then
	bun_version="$("$BUN_BIN" --version)"
	if version_ge "$bun_version" "$EXPECTED_BUN_MIN"; then
		pass "Bun version $bun_version (need >= $EXPECTED_BUN_MIN)"
	else
		fail "Bun version $bun_version is too old (need >= $EXPECTED_BUN_MIN)."
	fi
fi

if command -v "$PODMAN_BIN" >/dev/null 2>&1; then
	if "$PODMAN_BIN" info >/dev/null 2>&1; then
		pass "Podman daemonless runtime is reachable"
	else
		fail "Podman is installed but \`podman info\` failed."
	fi

	if "$PODMAN_BIN" compose version >/dev/null 2>&1; then
		pass "Podman Compose is available"
	else
		fail "Podman Compose is not available. Install or enable \`podman compose\`."
	fi

	socket_path="${PODMAN_SOCKET:-}"
	if [[ -z "$socket_path" && -n "${XDG_RUNTIME_DIR:-}" ]]; then
		socket_path="${XDG_RUNTIME_DIR}/podman/podman.sock"
	fi

	if [[ -z "$socket_path" ]]; then
		fail "Could not determine the Podman socket path. Set PODMAN_SOCKET or XDG_RUNTIME_DIR."
	elif [[ -S "$socket_path" ]]; then
		pass "Podman socket detected at $socket_path"
	else
		fail "Podman socket missing at $socket_path. Start it with \`systemctl --user start podman.socket\`."
	fi

	if "$PODMAN_BIN" network inspect stageflow_net >/dev/null 2>&1; then
		pass "Podman network \`stageflow_net\` already exists"
	else
		warn "Podman network \`stageflow_net\` is missing. \`just setup\` will create it."
	fi

	check_image "localhost/stageflow/extractor:latest"
	check_image "localhost/stageflow/scanner-runner:latest"
fi

if [[ -f "$ENV_FILE" ]]; then
	pass "Found $(basename "$ENV_FILE")"

	vite_api_url="$(read_env_value VITE_API_URL || true)"
	vite_site_url="$(read_env_value VITE_SITE_URL || true)"
	public_domain="$(read_env_value STAGEFLOW_PUBLIC_DOMAIN || true)"

	if [[ -n "$vite_api_url" && ! "$vite_api_url" =~ ^http://(localhost|127\.0\.0\.1)(:[0-9]+)?$ ]]; then
		warn "$(basename "$ENV_FILE") still points VITE_API_URL at a hosted endpoint. \`just demo\` overrides this for the containerized demo, but \`just run clients/web\` and \`stageflow project\` expect http://localhost:8080 for repo-local work."
	fi

	if [[ -n "$vite_site_url" && ! "$vite_site_url" =~ ^http://(localhost|127\.0\.0\.1)(:[0-9]+)?$ ]]; then
		warn "$(basename "$ENV_FILE") still points VITE_SITE_URL at a hosted endpoint. For repo-local work, prefer http://localhost:3000."
	fi

	if [[ -n "$public_domain" && "$public_domain" != "localhost" ]]; then
		warn "$(basename "$ENV_FILE") still uses STAGEFLOW_PUBLIC_DOMAIN=$public_domain. For repo-local work, prefer localhost."
	fi
else
	fail "Missing $(basename "$ENV_FILE"). Copy .env.example to .env before starting the stack."
fi

say ""

if (( fatal_count > 0 )); then
	say "StageFlow local diagnose found $fatal_count blocking issue(s) and $warn_count warning(s)."
	exit 1
fi

say "StageFlow local diagnose passed with $warn_count warning(s)."
say "Fastest local smoke test:"
say "  just demo"
say ""
say "Manual bootstrap:"
say "  just setup"
say "  just images"
say "  just dev up"
say "  just dev init"
