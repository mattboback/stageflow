#!/usr/bin/env bash
set -euo pipefail

# Renders the production Quadlet units from the *.in templates next to this
# script. The control plane in /home/operator/Deployment runs it on every deploy,
# restart and drift check; production secrets live outside the source tree.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_DIR="${SCRIPT_DIR}"
OUTPUT_DIR="${QUADLET_DIR:-${HOME}/.config/containers/systemd}"
USER_SYSTEMD_DIR="${USER_SYSTEMD_DIR:-${HOME}/.config/systemd/user}"
SKIP_STAGEFLOW_TARGET_BACKUP="${SKIP_STAGEFLOW_TARGET_BACKUP:-0}"

# REPO_ROOT is the checkout whose files the units bind-mount (scanner config).
REPO_ROOT="${STAGEFLOW_REPO_ROOT:-$(cd "${SCRIPT_DIR}/../.." && pwd)}"

ENV_FILE="${ENV_FILE:-${XDG_CONFIG_HOME:-${HOME}/.config}/stageflow/production.env}"
APP_ENV_FILE="${APP_ENV_FILE:-${ENV_FILE}}"
NETWORK_NAME="${STAGEFLOW_NETWORK_NAME:-${NETWORK_NAME:-stageflow_net}}"
HOST_UID="$(id -u)"
HOST_GID="$(id -g)"
SCANNER_OVERRIDES_EXAMPLE="${REPO_ROOT}/infra/scanners/scanners.example.yaml"
SCANNER_OVERRIDES="${REPO_ROOT}/infra/scanners/scanners.yaml"

# Units run the images tagged IMAGE_TAG. The control plane passes the commit
# SHA it built; without one, render what is deployed now (the state file's
# current= line, which the control plane writes).
STATE_FILE="${STAGEFLOW_STATE_FILE:-${XDG_STATE_HOME:-${HOME}/.local/state}/stageflow/image-tags}"
if [[ -z "${IMAGE_TAG:-}" && -f "${STATE_FILE}" ]]; then
  IMAGE_TAG="$(sed -n 's/^current=//p' "${STATE_FILE}")"
fi
IMAGE_TAG="${IMAGE_TAG:-}"

if [[ ! "${IMAGE_TAG}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  echo "quadlet-install: set IMAGE_TAG, or record current=<tag> in ${STATE_FILE}" >&2
  exit 1
fi

if [[ ! -f "${REPO_ROOT}/justfile" ]]; then
  echo "quadlet-install: cannot find stageflow repo at ${REPO_ROOT}" >&2
  echo "Set STAGEFLOW_REPO_ROOT to the stageflow application directory." >&2
  exit 1
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "quadlet-install: missing env file: ${ENV_FILE}" >&2
  exit 1
fi

if [[ ! -f "${APP_ENV_FILE}" ]]; then
  echo "quadlet-install: missing app env file: ${APP_ENV_FILE}" >&2
  exit 1
fi

if [[ ! -f "${SCANNER_OVERRIDES}" ]]; then
  if [[ ! -f "${SCANNER_OVERRIDES_EXAMPLE}" ]]; then
    echo "quadlet-install: missing scanner overrides example: ${SCANNER_OVERRIDES_EXAMPLE}" >&2
    exit 1
  fi

  mkdir -p "$(dirname "${SCANNER_OVERRIDES}")"
  cp "${SCANNER_OVERRIDES_EXAMPLE}" "${SCANNER_OVERRIDES}"
  echo "Created ${SCANNER_OVERRIDES} from ${SCANNER_OVERRIDES_EXAMPLE}"
fi

mkdir -p "${OUTPUT_DIR}"

# Prune: a rendered StageFlow unit whose template is gone is stopped, unmasked
# and deleted, so removing a template removes the service.
for rendered in "${OUTPUT_DIR}"/stageflow*.container "${OUTPUT_DIR}"/stageflow*.pod; do
  [[ -f "${rendered}" ]] || continue
  filename="$(basename "${rendered}")"
  [[ -f "${TEMPLATE_DIR}/${filename}.in" ]] && continue

  case "${filename}" in
    *.container) service="${filename%.container}.service" ;;
    *.pod) service="${filename%.pod}-pod.service" ;;
  esac
  if command -v systemctl >/dev/null 2>&1; then
    systemctl --user stop "${service}" >/dev/null 2>&1 || true
    systemctl --user reset-failed "${service}" >/dev/null 2>&1 || true
    systemctl --user unmask "${service}" >/dev/null 2>&1 || true
  fi
  rm -f "${rendered}"
  echo "Pruned ${rendered} (no template)"
done

escape_sed() {
  printf '%s' "$1" | sed -e 's/[&|]/\\&/g'
}

REPO_ROOT_ESC="$(escape_sed "${REPO_ROOT}")"
ENV_FILE_ESC="$(escape_sed "${ENV_FILE}")"
APP_ENV_FILE_ESC="$(escape_sed "${APP_ENV_FILE}")"
NETWORK_NAME_ESC="$(escape_sed "${NETWORK_NAME}")"
HOST_UID_ESC="$(escape_sed "${HOST_UID}")"
HOST_GID_ESC="$(escape_sed "${HOST_GID}")"
IMAGE_TAG_ESC="$(escape_sed "${IMAGE_TAG}")"

for template in "${TEMPLATE_DIR}"/*.in; do
  filename="$(basename "${template}")"

  output_path="${OUTPUT_DIR}/${filename%.in}"
  sed \
    -e "s|@REPO_ROOT@|${REPO_ROOT_ESC}|g" \
    -e "s|@ENV_FILE@|${ENV_FILE_ESC}|g" \
    -e "s|@APP_ENV_FILE@|${APP_ENV_FILE_ESC}|g" \
    -e "s|@NETWORK_NAME@|${NETWORK_NAME_ESC}|g" \
    -e "s|@HOST_UID@|${HOST_UID_ESC}|g" \
    -e "s|@HOST_GID@|${HOST_GID_ESC}|g" \
    -e "s|@IMAGE_TAG@|${IMAGE_TAG_ESC}|g" \
    "${template}" > "${output_path}"
  echo "Installed ${output_path}"
done

# systemd prefers unit files from ~/.config/systemd/user over quadlet-generated targets.
# Make the repo authoritative by also installing stageflow.target there.
USER_TARGET="${USER_SYSTEMD_DIR}/stageflow.target"
RENDERED_TARGET="${OUTPUT_DIR}/stageflow.target"

if [[ ! -f "${RENDERED_TARGET}" ]]; then
  echo "quadlet-install: missing rendered target: ${RENDERED_TARGET}" >&2
  exit 1
fi

mkdir -p "${USER_SYSTEMD_DIR}"

if ! cmp -s "${RENDERED_TARGET}" "${USER_TARGET}" 2>/dev/null; then
  if [[ "${SKIP_STAGEFLOW_TARGET_BACKUP}" != "1" && -f "${USER_TARGET}" ]]; then
    timestamp="$(date +%Y%m%d_%H%M%S)"
    backup_dir="${USER_SYSTEMD_DIR}/_backup_stageflow_${timestamp}"
    mkdir -p "${backup_dir}"
    cp "${USER_TARGET}" "${backup_dir}/stageflow.target"
    echo "Backed up ${USER_TARGET} -> ${backup_dir}/stageflow.target"
  fi

  cp "${RENDERED_TARGET}" "${USER_TARGET}"
  echo "Installed ${USER_TARGET}"
else
  echo "stageflow.target unchanged: ${USER_TARGET}"
fi

if command -v systemctl >/dev/null 2>&1; then
  systemctl --user daemon-reload
fi
