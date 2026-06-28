#!/usr/bin/env bash
set -Eeuo pipefail

PODMAN="${PODMAN:-podman}"

required_images=(
  "localhost/stageflow/platform-api:latest"
  "localhost/stageflow/orchestrator:latest"
  "localhost/stageflow/frontend-react:latest"
  "localhost/stageflow/extractor:latest"
  "localhost/stageflow/scanner-runner:latest"
)

missing=()

for image in "${required_images[@]}"; do
  if ! "$PODMAN" image exists "$image"; then
    missing+=("$image")
  fi
done

if (( ${#missing[@]} > 0 )); then
  echo "[images] Missing required image(s):" >&2
  printf '  - %s\n' "${missing[@]}" >&2
  exit 1
fi

echo "[images] Verified local StageFlow image inventory."
