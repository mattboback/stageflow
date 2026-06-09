#!/usr/bin/env bash
set -euo pipefail

# Host launcher: provisions required MinIO buckets, app policy, and app user by
# running provision.sh inside the official MinIO client container. The actual
# provisioning logic lives in provision.sh so that the prod stageflow-minio-init
# Quadlet and this host launcher share a single source of truth.

PODMAN=${PODMAN:-podman}
MC_IMAGE=${MC_IMAGE:-docker.io/minio/mc}
MC_NETWORK=${MC_NETWORK:-host}
MINIO_ENDPOINT=${MINIO_ENDPOINT:-http://127.0.0.1:9000}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -f .env ]]; then
	set -a
	# shellcheck source=/dev/null
	source .env
	set +a
fi

: "${MINIO_ROOT_USER:?MINIO_ROOT_USER must be set, or provide it in .env}"
: "${MINIO_ROOT_PASSWORD:?MINIO_ROOT_PASSWORD must be set, or provide it in .env}"

exec "$PODMAN" run --rm --network "$MC_NETWORK" \
	-e MINIO_ENDPOINT="$MINIO_ENDPOINT" \
	-e MINIO_ROOT_USER="$MINIO_ROOT_USER" \
	-e MINIO_ROOT_PASSWORD="$MINIO_ROOT_PASSWORD" \
	-e MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-}" \
	-e MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-}" \
	-e MINIO_APP_POLICY="${MINIO_APP_POLICY:-stageflow-app}" \
	-e MINIO_ALIAS="${MINIO_ALIAS:-stageflow}" \
	-v "${SCRIPT_DIR}/provision.sh:/usr/local/bin/provision.sh:ro,z" \
	--entrypoint /usr/local/bin/provision.sh \
	"$MC_IMAGE"
