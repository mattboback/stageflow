#!/usr/bin/env bash
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

# Explicit shell settings (including deploy SHA tags and demo URLs) take
# precedence over the selected environment file, matching Compose and Just.
declare -A env_overrides=()
for name in PODMAN IMAGE_TAG PODMAN_BUILD_NETWORK PODMAN_BUILD_FORMAT PODMAN_BUILD_NO_CACHE \
  VITE_API_URL VITE_SITE_URL VITE_DEPLOYMENT_MODE \
  PLATFORM_API_PORT FRONTEND_PORT; do
  if [[ -v "$name" ]]; then
    env_overrides["$name"]="${!name}"
  fi
done

ENV_FILE="${ENV_FILE:-$REPO_ROOT/.env}"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi
for name in "${!env_overrides[@]}"; do
  export "$name=${env_overrides[$name]}"
done

PODMAN="${PODMAN:-podman}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
PODMAN_BUILD_NETWORK="${PODMAN_BUILD_NETWORK:-host}"
PODMAN_BUILD_FORMAT="${PODMAN_BUILD_FORMAT:-docker}"
PODMAN_BUILD_NO_CACHE="${PODMAN_BUILD_NO_CACHE:-0}"
PODMAN_BUILD_ARGS=(--format "$PODMAN_BUILD_FORMAT" --network "$PODMAN_BUILD_NETWORK")

if [[ ! "$IMAGE_TAG" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  echo "[images] invalid IMAGE_TAG: $IMAGE_TAG" >&2
  exit 1
fi
if [[ "$PODMAN_BUILD_NO_CACHE" == "1" ]]; then
  PODMAN_BUILD_ARGS+=(--no-cache)
fi
if [[ ! -f "$REPO_ROOT/justfile" ]]; then
  echo "[images] repo root resolution failed: expected $REPO_ROOT/justfile" >&2
  exit 1
fi

build() {
  local label="$1"
  local image="$2"
  local dockerfile="$3"
  shift 3

  echo "[images] -> $label ($dockerfile)"
  "$PODMAN" build "${PODMAN_BUILD_ARGS[@]}" -t "localhost/stageflow/$image:$IMAGE_TAG" -f "$dockerfile" "$REPO_ROOT" "$@"
}

echo "[images] Building Go services (tag $IMAGE_TAG)..."
build platform-api platform-api services/platform-api/Dockerfile
build orchestrator orchestrator services/orchestrator/Dockerfile

echo "[images] Building React frontend..."
build frontend-react frontend-react clients/web/Dockerfile \
  --build-arg VITE_API_URL="${VITE_API_URL:-http://localhost:${PLATFORM_API_PORT:-8080}}" \
  --build-arg VITE_SITE_URL="${VITE_SITE_URL:-http://localhost:${FRONTEND_PORT:-3000}}" \
  --build-arg VITE_DEPLOYMENT_MODE="${VITE_DEPLOYMENT_MODE:-self-hosted}"

echo "[images] Building job images..."
build archive-extractor extractor services/archive-extractor/Dockerfile
echo "[images] -> scanner-runner (services/scanner-runner/Dockerfile)"
echo "[images]    first-time builds download Playwright Chromium and take the longest"
"$PODMAN" build \
  "${PODMAN_BUILD_ARGS[@]}" \
  --ignorefile services/scanner-runner/.dockerignore \
  -t "localhost/stageflow/scanner-runner:$IMAGE_TAG" \
  -f services/scanner-runner/Dockerfile \
  "$REPO_ROOT"

IMAGE_TAG="$IMAGE_TAG" ./infra/scripts/verify-image-inventory.sh

echo "[images] Done."
