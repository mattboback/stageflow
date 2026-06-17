#!/usr/bin/env bash
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

PODMAN="${PODMAN:-podman}"
ENV_FILE="${ENV_FILE:-$REPO_ROOT/.env}"
PODMAN_BUILD_NETWORK="${PODMAN_BUILD_NETWORK:-host}"
PODMAN_BUILD_FORMAT="${PODMAN_BUILD_FORMAT:-docker}"
PODMAN_BUILD_NO_CACHE="${PODMAN_BUILD_NO_CACHE:-0}"
PODMAN_BUILD_ARGS=(--format "$PODMAN_BUILD_FORMAT" --network "$PODMAN_BUILD_NETWORK")

if [[ "$PODMAN_BUILD_NO_CACHE" == "1" ]]; then
  PODMAN_BUILD_ARGS+=(--no-cache)
fi

if [[ ! -f "$REPO_ROOT/justfile" ]]; then
  echo "[images] repo root resolution failed: expected $REPO_ROOT/justfile" >&2
  exit 1
fi

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

build() {
  local label="$1"
  local primary_tag="$2"
  local compat_tag="$3"
  local dockerfile="$4"
  shift 4

  echo "[images] -> $label ($dockerfile)"
  "$PODMAN" build "${PODMAN_BUILD_ARGS[@]}" -t "$primary_tag" -f "$dockerfile" "$REPO_ROOT" "$@"
  "$PODMAN" tag "$primary_tag" "$compat_tag"
}

echo "[images] Building Go services..."
build platform-api localhost/stageflow/platform-api:latest stageflow/platform-api:latest services/platform-api/Dockerfile
build orchestrator localhost/stageflow/orchestrator:latest stageflow/orchestrator:latest services/orchestrator/Dockerfile

echo "[images] Building React frontend..."
build frontend-react localhost/stageflow/frontend-react:latest stageflow/frontend-react:latest clients/web-react/Dockerfile \
  --build-arg VITE_API_URL="${VITE_API_URL:-http://localhost:8080}" \
  --build-arg VITE_SITE_TITLE="${VITE_SITE_TITLE:-StageFlow}" \
  --build-arg VITE_SITE_URL="${VITE_SITE_URL:-http://localhost:3020}" \
  --build-arg VITE_GITHUB_URL="${VITE_GITHUB_URL:-https://github.com/mattboback/stageflow}" \
  --build-arg VITE_TAGLINE="${VITE_TAGLINE:-Podman-native web accessibility and quality scanning platform}"

echo "[images] Building job images..."
build archive-extractor localhost/stageflow/extractor:latest stageflow/extractor:latest services/archive-extractor/Dockerfile
echo "[images] -> scanner-runner (services/scanner-runner/Dockerfile)"
echo "[images]    first-time builds download Playwright Chromium and take the longest"
"$PODMAN" build \
  "${PODMAN_BUILD_ARGS[@]}" \
  --ignorefile services/scanner-runner/.dockerignore \
  -t localhost/stageflow/scanner-runner:latest \
  -f services/scanner-runner/Dockerfile \
  "$REPO_ROOT"
"$PODMAN" tag localhost/stageflow/scanner-runner:latest stageflow/scanner-runner:latest

./infra/scripts/verify-image-inventory.sh

echo "[images] Done."
