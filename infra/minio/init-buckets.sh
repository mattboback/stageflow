#!/usr/bin/env bash
set -euo pipefail

# Initializes required MinIO buckets using the official MinIO client container.

PODMAN=${PODMAN:-podman}
MC_IMAGE=${MC_IMAGE:-docker.io/minio/mc}
MC_NETWORK=${MC_NETWORK:-host}
MINIO_ENDPOINT=${MINIO_ENDPOINT:-http://127.0.0.1:9000}

if [[ -f .env ]]; then
	set -a
	# shellcheck source=/dev/null
	source .env
	set +a
fi

: "${MINIO_ROOT_USER:?MINIO_ROOT_USER must be set, or provide it in .env}"
: "${MINIO_ROOT_PASSWORD:?MINIO_ROOT_PASSWORD must be set, or provide it in .env}"

MINIO_ACCESS_KEY=${MINIO_ACCESS_KEY:-}
MINIO_SECRET_KEY=${MINIO_SECRET_KEY:-}
MINIO_APP_POLICY=${MINIO_APP_POLICY:-stageflow-app}
MINIO_ALIAS=${MINIO_ALIAS:-stageflow}
BUCKETS=("scanner-staging" "scanner-artifacts")

# Build MC_HOST style URL: scheme://accessKey:secretKey@host:port
build_mc_host() {
	local endpoint="$1"
	local scheme hostport

	if [[ "$endpoint" == http://* ]]; then
		scheme="http://"
		hostport="${endpoint#http://}"
	elif [[ "$endpoint" == https://* ]]; then
		scheme="https://"
		hostport="${endpoint#https://}"
	else
		# default to http if no scheme
		scheme="http://"
		hostport="$endpoint"
	fi

	printf "%s%s:%s@%s" "$scheme" "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" "$hostport"
}

MC_HOST_VALUE="$(build_mc_host "$MINIO_ENDPOINT")"

# Run mc with given arguments against the configured endpoint.
run_mc() {
	# mc image uses 'mc' as entrypoint, so we just pass subcommands/args.
	local volume_args=()
	if [[ -n "${MC_VOLUME:-}" ]]; then
		volume_args=(-v "$MC_VOLUME")
	fi

	$PODMAN run --rm --network "$MC_NETWORK" \
		"${volume_args[@]}" \
		-e MC_HOST_"${MINIO_ALIAS}"="${MC_HOST_VALUE}" \
		"$MC_IMAGE" "$@"
}

echo "[minio] Ensuring buckets exist via mc container..."

for bucket in "${BUCKETS[@]}"; do
	if run_mc ls "${MINIO_ALIAS}/${bucket}" >/dev/null 2>&1; then
		echo "[minio] Bucket ${bucket} already present"
		continue
	fi
	echo "[minio] Creating bucket ${bucket}"
	run_mc mb "${MINIO_ALIAS}/${bucket}"
done

echo "[minio] Ensuring anonymous access is disabled on scanner-artifacts"
run_mc anonymous set none "${MINIO_ALIAS}/scanner-artifacts"
echo "[minio] Ensuring anonymous access is disabled on scanner-staging"
run_mc anonymous set none "${MINIO_ALIAS}/scanner-staging"

if [[ -n "${MINIO_ACCESS_KEY}" && -n "${MINIO_SECRET_KEY}" ]]; then
	if [[ "${MINIO_ACCESS_KEY}" == "${MINIO_ROOT_USER}" ]]; then
		echo "[minio] MINIO_ACCESS_KEY matches MINIO_ROOT_USER; refusing to manage app user" >&2
		exit 1
	fi

	policy_dir="$(mktemp -d)"
	policy_file="$policy_dir/${MINIO_APP_POLICY}.json"
	trap 'rm -rf "$policy_dir"' EXIT

	cat >"$policy_file" <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetBucketLocation",
        "s3:ListBucket",
        "s3:ListBucketMultipartUploads"
      ],
      "Resource": [
        "arn:aws:s3:::scanner-staging",
        "arn:aws:s3:::scanner-artifacts"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:AbortMultipartUpload",
        "s3:ListMultipartUploadParts"
      ],
      "Resource": [
        "arn:aws:s3:::scanner-staging/*",
        "arn:aws:s3:::scanner-artifacts/*"
      ]
    }
  ]
}
EOF

	if run_mc admin user info "${MINIO_ALIAS}" "${MINIO_ACCESS_KEY}" >/dev/null 2>&1; then
		run_mc admin user remove "${MINIO_ALIAS}" "${MINIO_ACCESS_KEY}" >/dev/null
	fi

	if run_mc admin policy info "${MINIO_ALIAS}" "${MINIO_APP_POLICY}" >/dev/null 2>&1; then
		run_mc admin policy remove "${MINIO_ALIAS}" "${MINIO_APP_POLICY}" >/dev/null
	fi

	echo "[minio] Creating app policy ${MINIO_APP_POLICY}"
	MC_VOLUME="${policy_dir}:/policy:ro,z" \
		run_mc admin policy create "${MINIO_ALIAS}" "${MINIO_APP_POLICY}" "/policy/${MINIO_APP_POLICY}.json"

	echo "[minio] Creating app user ${MINIO_ACCESS_KEY}"
	run_mc admin user add "${MINIO_ALIAS}" "${MINIO_ACCESS_KEY}" "${MINIO_SECRET_KEY}"
	run_mc admin policy attach "${MINIO_ALIAS}" "${MINIO_APP_POLICY}" --user "${MINIO_ACCESS_KEY}"
else
	echo "[minio] MINIO_ACCESS_KEY/SECRET_KEY not set; skipping app user/policy"
fi

echo "[minio] Buckets ready."
