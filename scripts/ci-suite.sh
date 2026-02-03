#!/usr/bin/env bash
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

SUITE_FILE="${1:-${SUITE_FILE:-tools/suite-runner/suite.sample.yml}}"

ENV_FILE="${ENV_FILE:-$REPO_ROOT/.env}"
if [[ ! -f "$ENV_FILE" ]]; then
	ENV_FILE="$REPO_ROOT/.env.example"
fi

if [[ -f "$ENV_FILE" ]]; then
	set -a
	# shellcheck disable=SC1090
	source "$ENV_FILE"
	set +a
fi

API_BASE_URL="${PLATFORM_API_BASE_URL:-http://127.0.0.1:8080}"
HEALTH_URL="${API_BASE_URL%/}/healthz"

COMPOSE_FILES=(
	-f infra/compose/podman-compose.yml
	-f infra/compose/podman-compose.test.yml
)

compose_up() {
	if command -v podman-compose >/dev/null 2>&1; then
		podman-compose "${COMPOSE_FILES[@]}" up -d
		return
	fi

	if command -v podman >/dev/null 2>&1; then
		podman compose "${COMPOSE_FILES[@]}" up -d
		return
	fi

	echo "ci-suite: podman-compose or podman compose is required" >&2
	exit 127
}

compose_down() {
	if command -v podman-compose >/dev/null 2>&1; then
		podman-compose "${COMPOSE_FILES[@]}" down --remove-orphans
		return
	fi

	if command -v podman >/dev/null 2>&1; then
		podman compose "${COMPOSE_FILES[@]}" down --remove-orphans
		return
	fi
}

KEEP_RUNNING="${SUITE_KEEP_RUNNING:-0}"
if [[ "$KEEP_RUNNING" != "1" ]]; then
	trap compose_down EXIT
fi

echo "[ci-suite] starting stack via podman compose overlays"
compose_up

ATTEMPTS="${SUITE_HEALTH_ATTEMPTS:-60}"
SLEEP_SEC="${SUITE_HEALTH_SLEEP:-2}"

echo "[ci-suite] waiting for platform-api health: ${HEALTH_URL}"
for ((i = 1; i <= ATTEMPTS; i++)); do
	if curl -fsS --max-time 5 "$HEALTH_URL" >/dev/null; then
		echo "[ci-suite] platform-api healthy"
		break
	fi

	if [[ "$i" -eq "$ATTEMPTS" ]]; then
		echo "[ci-suite] timed out waiting for platform-api health" >&2
		exit 1
	fi

	sleep "$SLEEP_SEC"
done

echo "[ci-suite] running suite: ${SUITE_FILE}"
if command -v suite-runner >/dev/null 2>&1; then
	suite-runner -suite "$SUITE_FILE" -api "$API_BASE_URL"
else
	go run ./tools/suite-runner -suite "$SUITE_FILE" -api "$API_BASE_URL"
fi

