#!/usr/bin/env bash
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

PODMAN="${PODMAN:-podman}"
ENV_FILE="${ENV_FILE:-$REPO_ROOT/.env}"
PODMAN_BUILD_NETWORK="${PODMAN_BUILD_NETWORK:-host}"

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
  local primary_tag="$1"
  local compat_tag="$2"
  local dockerfile="$3"
  shift 3

  "$PODMAN" build --network "$PODMAN_BUILD_NETWORK" -t "$primary_tag" -f "$dockerfile" "$REPO_ROOT" "$@"
  "$PODMAN" tag "$primary_tag" "$compat_tag"
}

echo "[images] Building Go services..."
build localhost/stageflow/platform-api:latest stageflow/platform-api:latest services/platform-api/Dockerfile
build localhost/stageflow/orchestrator:latest stageflow/orchestrator:latest services/orchestrator/Dockerfile

echo "[images] Building frontend (SvelteKit)..."
build localhost/stageflow/frontend:latest stageflow/frontend:latest clients/web/Dockerfile \
  --build-arg VITE_API_URL="${VITE_API_URL:-https://example.com}" \
  --build-arg VITE_SITE_TITLE="${VITE_SITE_TITLE:-StageFlow}" \
  --build-arg VITE_SITE_URL="${VITE_SITE_URL:-https://example.com}" \
  --build-arg VITE_GITHUB_URL="${VITE_GITHUB_URL:-https://github.com/your-handle}" \
  --build-arg VITE_TAGLINE="${VITE_TAGLINE:-Podman-native web accessibility scanning platform}" \
  --build-arg VITE_AI_NAVIGATOR_DEFAULT_MODEL="${VITE_AI_NAVIGATOR_DEFAULT_MODEL:-openai/gpt-4o-mini}"

echo "[images] Building job images..."
build localhost/stageflow/extractor:latest stageflow/extractor:latest services/archive-extractor/Dockerfile
"$PODMAN" build \
  --network "$PODMAN_BUILD_NETWORK" \
  --ignorefile services/scanner-runner/.dockerignore \
  -t localhost/stageflow/scanner-runner:latest \
  -f services/scanner-runner/Dockerfile \
  "$REPO_ROOT"
"$PODMAN" tag localhost/stageflow/scanner-runner:latest stageflow/scanner-runner:latest

echo "[images] Done."
