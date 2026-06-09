#!/bin/sh
set -eu

# Provisions MinIO buckets, app policy, and app user. Runs *inside* the official
# minio/mc container (mc on PATH). Used both by the prod stageflow-minio-init
# Quadlet and by the host launcher init-buckets.sh. Idempotent and safe to re-run.

MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://127.0.0.1:9000}"
MINIO_ALIAS="${MINIO_ALIAS:-stageflow}"
MINIO_APP_POLICY="${MINIO_APP_POLICY:-stageflow-app}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-}"

: "${MINIO_ROOT_USER:?MINIO_ROOT_USER must be set}"
: "${MINIO_ROOT_PASSWORD:?MINIO_ROOT_PASSWORD must be set}"

case "$MINIO_ENDPOINT" in
	http://* | https://*) url="$MINIO_ENDPOINT" ;;
	*) url="http://$MINIO_ENDPOINT" ;;
esac

echo "[minio] Configuring mc alias ${MINIO_ALIAS} -> ${url}"
mc alias set "$MINIO_ALIAS" "$url" "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null

echo "[minio] Waiting for MinIO to become ready..."
ready=""
i=1
while [ "$i" -le 60 ]; do
	if mc ready "$MINIO_ALIAS" >/dev/null 2>&1; then
		ready=1
		break
	fi
	sleep 2
	i=$((i + 1))
done
if [ -z "$ready" ]; then
	echo "[minio] MinIO did not become ready in time" >&2
	exit 1
fi

for bucket in scanner-staging scanner-artifacts; do
	if mc ls "${MINIO_ALIAS}/${bucket}" >/dev/null 2>&1; then
		echo "[minio] Bucket ${bucket} already present"
	else
		echo "[minio] Creating bucket ${bucket}"
		mc mb "${MINIO_ALIAS}/${bucket}"
	fi
	echo "[minio] Ensuring anonymous access is disabled on ${bucket}"
	mc anonymous set none "${MINIO_ALIAS}/${bucket}" >/dev/null
done

if [ -z "$MINIO_ACCESS_KEY" ] || [ -z "$MINIO_SECRET_KEY" ]; then
	echo "[minio] MINIO_ACCESS_KEY/SECRET_KEY not set; skipping app user/policy"
	echo "[minio] Buckets ready."
	exit 0
fi

if [ "$MINIO_ACCESS_KEY" = "$MINIO_ROOT_USER" ]; then
	echo "[minio] MINIO_ACCESS_KEY matches MINIO_ROOT_USER; refusing to manage app user" >&2
	exit 1
fi

policy_file="$(mktemp)"
trap 'rm -f "$policy_file"' EXIT
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

# Upsert the app user first: `mc admin user add` updates the secret if the user
# already exists, so this both creates it and repairs a rotated secret.
echo "[minio] Ensuring app user ${MINIO_ACCESS_KEY}"
mc admin user add "$MINIO_ALIAS" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY"

# Refresh the policy from desired content. Detach it from the user first so the
# remove does not fail with "policy in use" on re-runs against a live stack.
mc admin policy detach "$MINIO_ALIAS" "$MINIO_APP_POLICY" --user "$MINIO_ACCESS_KEY" >/dev/null 2>&1 || true
if mc admin policy info "$MINIO_ALIAS" "$MINIO_APP_POLICY" >/dev/null 2>&1; then
	mc admin policy remove "$MINIO_ALIAS" "$MINIO_APP_POLICY" >/dev/null
fi
echo "[minio] Creating app policy ${MINIO_APP_POLICY}"
mc admin policy create "$MINIO_ALIAS" "$MINIO_APP_POLICY" "$policy_file"
mc admin policy attach "$MINIO_ALIAS" "$MINIO_APP_POLICY" --user "$MINIO_ACCESS_KEY"

echo "[minio] Buckets ready."
