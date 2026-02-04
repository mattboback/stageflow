#!/usr/bin/env bash
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

PODMAN="${PODMAN:-podman}"
ENV_FILE="${ENV_FILE:-$REPO_ROOT/.env}"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

build() {
  local tag="$1"
  local dockerfile="$2"
  shift 2

  "$PODMAN" build -t "$tag" -f "$dockerfile" "$REPO_ROOT" "$@"
}

echo "[images] Building Go services..."
build stageflow/platform-api:latest platform/api/Dockerfile
build stageflow/orchestrator:latest platform/orchestrator/Dockerfile

echo "[images] Building frontend (SvelteKit)..."
build stageflow/frontend:latest frontend/Dockerfile \
  --build-arg VITE_API_URL="${VITE_API_URL:-https://example.com}" \
  --build-arg VITE_SITE_TITLE="${VITE_SITE_TITLE:-StageFlow}" \
  --build-arg VITE_SITE_URL="${VITE_SITE_URL:-https://example.com}" \
  --build-arg VITE_GITHUB_URL="${VITE_GITHUB_URL:-https://github.com/your-handle}" \
  --build-arg VITE_TAGLINE="${VITE_TAGLINE:-Podman-native web accessibility scanning platform}" \
  --build-arg VITE_AI_NAVIGATOR_DEFAULT_MODEL="${VITE_AI_NAVIGATOR_DEFAULT_MODEL:-openai/gpt-4o-mini}"

echo "[images] Building job images..."
build stageflow/extractor:latest platform/extractor/Dockerfile
"$PODMAN" build \
  --ignorefile platform/scanner-runner/.dockerignore \
  -t stageflow/scanner-runner:latest \
  -f platform/scanner-runner/Dockerfile \
  "$REPO_ROOT"

echo "[images] Done."
