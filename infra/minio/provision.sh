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
MINIO_STAGING_RETENTION_DAYS="${MINIO_STAGING_RETENTION_DAYS:-1}"
MINIO_ARTIFACT_RETENTION_DAYS="${MINIO_ARTIFACT_RETENTION_DAYS:-1}"
MINIO_APPLY_LIFECYCLES="${MINIO_APPLY_LIFECYCLES:-true}"

: "${MINIO_ROOT_USER:?MINIO_ROOT_USER must be set}"
: "${MINIO_ROOT_PASSWORD:?MINIO_ROOT_PASSWORD must be set}"

validate_retention_days() {
	name="$1"
	value="$2"

	case "$value" in
		'' | *[!0-9]*)
			echo "[minio] ${name} must be a positive integer (got: ${value:-empty})" >&2
			exit 1
			;;
	esac

	if [ "$value" -le 0 ]; then
		echo "[minio] ${name} must be greater than zero (got: ${value})" >&2
		exit 1
	fi
}

validate_retention_days MINIO_STAGING_RETENTION_DAYS "$MINIO_STAGING_RETENTION_DAYS"
validate_retention_days MINIO_ARTIFACT_RETENTION_DAYS "$MINIO_ARTIFACT_RETENTION_DAYS"

case "$MINIO_APPLY_LIFECYCLES" in
	true | false) ;;
	*)
		echo "[minio] MINIO_APPLY_LIFECYCLES must be true or false (got: ${MINIO_APPLY_LIFECYCLES})" >&2
		exit 1
		;;
esac

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

temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT

write_lifecycle_config() {
	path="$1"
	rule_id="$2"
	days="$3"

	cat >"$path" <<EOF
{
  "Rules": [
    {
      "Expiration": { "Days": ${days} },
      "ID": "${rule_id}",
      "Status": "Enabled"
    }
  ]
}
EOF
}

staging_lifecycle_file="${temp_dir}/scanner-staging-lifecycle.json"
artifact_lifecycle_file="${temp_dir}/scanner-artifacts-lifecycle.json"
write_lifecycle_config "$staging_lifecycle_file" "stageflow-staging-retention" "$MINIO_STAGING_RETENTION_DAYS"
write_lifecycle_config "$artifact_lifecycle_file" "stageflow-artifact-retention" "$MINIO_ARTIFACT_RETENTION_DAYS"

for bucket in scanner-staging scanner-artifacts scanner-baselines; do
	if mc ls "${MINIO_ALIAS}/${bucket}" >/dev/null 2>&1; then
		echo "[minio] Bucket ${bucket} already present"
	else
		echo "[minio] Creating bucket ${bucket}"
		mc mb "${MINIO_ALIAS}/${bucket}"
	fi
	echo "[minio] Ensuring anonymous access is disabled on ${bucket}"
	mc anonymous set none "${MINIO_ALIAS}/${bucket}" >/dev/null
done

# Importing the complete desired lifecycle document replaces the existing
# StageFlow bucket policy instead of appending duplicate rules on every run.
if [ "$MINIO_APPLY_LIFECYCLES" = "true" ]; then
	echo "[minio] Enforcing ${MINIO_STAGING_RETENTION_DAYS}-day retention on scanner-staging"
	mc ilm rule import "${MINIO_ALIAS}/scanner-staging" <"$staging_lifecycle_file"
	echo "[minio] Enforcing ${MINIO_ARTIFACT_RETENTION_DAYS}-day retention on scanner-artifacts"
	mc ilm rule import "${MINIO_ALIAS}/scanner-artifacts" <"$artifact_lifecycle_file"
else
	echo "[minio] Lifecycle updates deferred; buckets and application policy only"
fi

if [ -z "$MINIO_ACCESS_KEY" ] || [ -z "$MINIO_SECRET_KEY" ]; then
	echo "[minio] MINIO_ACCESS_KEY/SECRET_KEY not set; skipping app user/policy"
	echo "[minio] Buckets ready."
	exit 0
fi

if [ "$MINIO_ACCESS_KEY" = "$MINIO_ROOT_USER" ]; then
	echo "[minio] MINIO_ACCESS_KEY matches MINIO_ROOT_USER; refusing to manage app user" >&2
	exit 1
fi

policy_file="${temp_dir}/app-policy.json"
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
        "arn:aws:s3:::scanner-artifacts",
        "arn:aws:s3:::scanner-baselines"
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
        "arn:aws:s3:::scanner-artifacts/*",
        "arn:aws:s3:::scanner-baselines/*"
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
