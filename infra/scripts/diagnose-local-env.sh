#!/usr/bin/env bash
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GO_BIN="${GO:-go}"
BUN_BIN="${BUN:-bun}"
NODE_BIN="${NODE:-node}"
PODMAN_BIN="${PODMAN:-podman}"
JUST_BIN="${JUST:-just}"
CURL_BIN="${CURL:-curl}"
ENV_FILE="${ENV_FILE:-$REPO_ROOT/.env}"
EXPECTED_GO_MIN="1.26.10"
EXPECTED_BUN_MIN="1.3.8"
EXPECTED_NODE_MIN_MAJOR="24"

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

	[[ -f "$ENV_FILE" ]] || return 1
	line="$(grep -E "^${key}=" "$ENV_FILE" | tail -n1 || true)"
	[[ -n "$line" ]] || return 1
	printf '%s' "${line#*=}"
}

port_in_use() {
	local port="$1"
	local socket

	if command -v ss >/dev/null 2>&1; then
		while IFS= read -r socket; do
			[[ "$socket" == *":$port" ]] && return 0
		done < <(ss -H -ltn 2>/dev/null | awk '{print $4}')
		return 1
	fi

	if command -v lsof >/dev/null 2>&1; then
		lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
		return
	fi

	return 2
}

# Match Compose precedence without sourcing an arbitrary environment file.
effective_env_value() {
	local key="$1"
	local fallback="$2"
	local value
	if [[ -v "$key" ]]; then
		printf '%s' "${!key}"
		return
	fi
	if ! value="$(read_env_value "$key")"; then
		printf '%s' "$fallback"
		return
	fi
	value="${value%\"}"
	value="${value#\"}"
	value="${value%\'}"
	value="${value#\'}"
	printf '%s' "$value"
}

check_default_ports() {
	local conflicts=0
	local entry
	local label
	local port
	local required_ports=(
		"$(effective_env_value FRONTEND_PORT 3000):frontend"
		"$(effective_env_value PLATFORM_API_PORT 8080):Platform API"
		"$(effective_env_value ORCHESTRATOR_PORT 8081):orchestrator"
	)

	if ! command -v ss >/dev/null 2>&1 && ! command -v lsof >/dev/null 2>&1; then
		warn "Cannot inspect local port listeners because neither \`ss\` nor \`lsof\` is installed."
		return
	fi

	for entry in "${required_ports[@]}"; do
		port="${entry%%:*}"
		label="${entry#*:}"

		if port_in_use "$port"; then
			warn "Port $port is already listening ($label). Stop the conflicting service before \`just demo\` or \`just dev up\`; otherwise setup may reach the wrong local service."
			conflicts=$((conflicts + 1))
		fi
	done

	if (( conflicts == 0 )); then
		pass "Configured local stack ports are free"
	fi
}

check_network_alias_conflicts() {
	local network="$1"
	local alias_format
	local aliases
	local container_id
	local container_name
	local conflict_count=0
	local service_aliases=(
		"postgres"
		"orchestrator"
		"platform-api"
		"frontend"
	)

	alias_format="{{range (index .NetworkSettings.Networks \"${network}\").Aliases}}{{println .}}{{end}}"

	while read -r container_id container_name; do
		[[ -n "$container_id" ]] || continue

		aliases="$("$PODMAN_BIN" inspect "$container_id" --format "$alias_format" 2>/dev/null || true)"
		[[ -n "$aliases" ]] || continue

		for service_alias in "${service_aliases[@]}"; do
			if grep -Fxq "$service_alias" <<<"$aliases"; then
				warn "Podman network \`${network}\` already has container \`${container_name}\` exposing service alias \`${service_alias}\`; use a different COMPOSE_PROJECT_NAME or STAGEFLOW_NETWORK_NAME before starting a second local stack."
				conflict_count=$((conflict_count + 1))
			fi
		done
	done < <("$PODMAN_BIN" ps -a --filter "network=$network" --format '{{.ID}} {{.Names}}' 2>/dev/null || true)

	if (( conflict_count == 0 )); then
		pass "No existing StageFlow service aliases detected on Podman network \`${network}\`"
	fi
}

COMPOSE_PROJECT_NAME="$(effective_env_value COMPOSE_PROJECT_NAME stageflow_dev)"
LOCAL_NETWORK="$(effective_env_value STAGEFLOW_NETWORK_NAME "${COMPOSE_PROJECT_NAME}_net")"
JOBS_NETWORK="$(effective_env_value STAGEFLOW_JOBS_NETWORK_NAME "${COMPOSE_PROJECT_NAME}_jobs")"

say "==> StageFlow local diagnose"

require_command "$GO_BIN" "Go"
require_command "$BUN_BIN" "Bun"
require_command "$NODE_BIN" "Node.js"
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

if command -v "$NODE_BIN" >/dev/null 2>&1; then
	node_version="$("$NODE_BIN" --version)"
	node_major="${node_version#v}"
	node_major="${node_major%%.*}"
	if [[ "$node_major" =~ ^[0-9]+$ ]] && (( node_major >= EXPECTED_NODE_MIN_MAJOR )); then
		pass "Node.js version $node_version (need >= $EXPECTED_NODE_MIN_MAJOR.x)"
	else
		fail "Node.js version $node_version is too old (need >= $EXPECTED_NODE_MIN_MAJOR.x). The repo pins Node $EXPECTED_NODE_MIN_MAJOR in .node-version; install it or set NODE=/path/to/node."
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

	if [[ -c /dev/net/tun ]]; then
		pass "/dev/net/tun is available for rootless Podman networking"
	else
		warn "/dev/net/tun is missing. Rootless Podman networking can fail with \`Failed to open() /dev/net/tun\` when using pasta or slirp4netns."
	fi

	if "$PODMAN_BIN" network inspect "$LOCAL_NETWORK" >/dev/null 2>&1; then
		pass "Podman network \`${LOCAL_NETWORK}\` already exists"
		check_network_alias_conflicts "$LOCAL_NETWORK"
	else
		warn "Podman network \`${LOCAL_NETWORK}\` is missing. \`just setup\` will create it."
	fi

	# Bridge-mode job pods need their own isolated network; the orchestrator
	# refuses to start on any other kind.
	if [[ "$(effective_env_value POD_NETNS_MODE bridge)" != "host" ]]; then
		job_network="$(effective_env_value POD_NETWORK "$JOBS_NETWORK")"
		if isolation="$("$PODMAN_BIN" network inspect "$job_network" --format '{{index .Options "isolate"}}' 2>/dev/null)"; then
			if [[ "$isolation" == "strict" ]]; then
				pass "Podman job network \`${job_network}\` exists and is isolated"
			else
				fail "Podman job network \`${job_network}\` is not isolated. Remove it and run \`just setup\`, which creates it with \`--opt isolate=strict\`."
			fi
		else
			warn "Podman job network \`${job_network}\` is missing. \`just setup\` will create it."
		fi
	fi

	image_tag="$(effective_env_value IMAGE_TAG latest)"
	check_image "localhost/stageflow/extractor:$image_tag"
	check_image "localhost/stageflow/scanner-runner:$image_tag"
fi

check_default_ports

if [[ "$(effective_env_value POD_NETNS_MODE bridge)" == "host" ]]; then
	pod_network="$(effective_env_value POD_NETWORK __unset__)"
	if [[ -n "$pod_network" ]]; then
		fail "Host-mode job pods require an explicitly empty POD_NETWORK= in .env."
	fi
fi

if [[ -f "$ENV_FILE" ]]; then
	pass "Found $(basename "$ENV_FILE")"

	vite_api_url="$(effective_env_value VITE_API_URL "")"
	vite_site_url="$(effective_env_value VITE_SITE_URL "")"
	public_domain="$(effective_env_value STAGEFLOW_PUBLIC_DOMAIN "")"

	if [[ -n "$vite_api_url" && ! "$vite_api_url" =~ ^http://(localhost|127\.0\.0\.1)(:[0-9]+)?$ ]]; then
		warn "$(basename "$ENV_FILE") still points VITE_API_URL at a hosted endpoint. For repo-local work, set VITE_API_URL to the configured API address; \`just demo\` uses this value unless STAGEFLOW_DEMO_VITE_API_URL overrides it."
	fi

	if [[ -n "$vite_site_url" && ! "$vite_site_url" =~ ^http://(localhost|127\.0\.0\.1)(:[0-9]+)?$ ]]; then
		warn "$(basename "$ENV_FILE") still points VITE_SITE_URL at a hosted endpoint. For repo-local work, set VITE_SITE_URL to the configured frontend address."
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
say "    guided wrapper using the local settings in .env"
say ""
say "Manual bootstrap:"
say "  just setup"
say "  just images"
say "  just dev up          # settings come from .env"

say "For localhost scans, set PLATFORM_API_ALLOW_PRIVATE_TARGETS=true, POD_NETNS_MODE=host, and POD_NETWORK= in .env."
