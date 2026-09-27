#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
cat > "$work/podman" <<'MOCK'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$BUILD_TEST_LOG"
MOCK
chmod +x "$work/podman"
cat > "$work/config.env" <<'ENV'
IMAGE_TAG=from-file
VITE_API_URL=http://localhost:28080
VITE_SITE_URL=http://localhost:23000
PODMAN_BUILD_NETWORK=bridge
ENV
export PODMAN="$work/podman" BUILD_TEST_LOG="$work/calls" ENV_FILE="$work/config.env"
IMAGE_TAG=explicit VITE_API_URL=http://localhost:18080 VITE_SITE_URL=http://localhost:13000 \
  PODMAN_BUILD_NETWORK=host "$root/infra/scripts/build-images.sh" > "$work/output"
rg -q -- '-t localhost/stageflow/platform-api:explicit' "$work/calls"
rg -q -- '--build-arg VITE_API_URL=http://localhost:18080' "$work/calls"
rg -q -- '--build-arg VITE_SITE_URL=http://localhost:13000' "$work/calls"
rg -q -- '--network host' "$work/calls"
if rg -q 'from-file|28080|23000|--network bridge' "$work/calls"; then
  echo 'build-images: environment file overrode explicit build settings' >&2
  exit 1
fi
: > "$work/calls"
env -u IMAGE_TAG -u VITE_API_URL -u VITE_SITE_URL -u PODMAN_BUILD_NETWORK \
  "$root/infra/scripts/build-images.sh" > "$work/output"
rg -q -- '-t localhost/stageflow/platform-api:from-file' "$work/calls"
rg -q -- '--build-arg VITE_API_URL=http://localhost:28080' "$work/calls"
rg -q -- '--network bridge' "$work/calls"
printf 'IMAGE_TAG=invalid/tag\n' > "$work/config.env"
: > "$work/calls"
if env -u IMAGE_TAG "$root/infra/scripts/build-images.sh" > "$work/output" 2>&1; then
  echo 'build-images: accepted invalid file image tag' >&2
  exit 1
fi
[[ ! -s "$work/calls" ]]
echo 'Image build environment tests passed.'
