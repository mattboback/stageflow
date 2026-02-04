#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEMPLATE_DIR="${REPO_ROOT}/infra/quadlets/templates"
OUTPUT_DIR="${QUADLET_DIR:-${HOME}/.config/containers/systemd}"

ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env}"
APP_ENV_FILE="${APP_ENV_FILE:-${ENV_FILE}}"
NETWORK_NAME="${STAGEFLOW_NETWORK_NAME:-${NETWORK_NAME:-stageflow_net}}"
SCANNER_OVERRIDES_EXAMPLE="${REPO_ROOT}/infra/scanners/scanners.example.yaml"
SCANNER_OVERRIDES="${REPO_ROOT}/infra/scanners/scanners.yaml"

if [[ ! -d "${TEMPLATE_DIR}" ]]; then
  echo "quadlet-install: missing templates directory: ${TEMPLATE_DIR}" >&2
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

escape_sed() {
  printf '%s' "$1" | sed -e 's/[&|]/\\&/g'
}

REPO_ROOT_ESC="$(escape_sed "${REPO_ROOT}")"
ENV_FILE_ESC="$(escape_sed "${ENV_FILE}")"
APP_ENV_FILE_ESC="$(escape_sed "${APP_ENV_FILE}")"
NETWORK_NAME_ESC="$(escape_sed "${NETWORK_NAME}")"

for template in "${TEMPLATE_DIR}"/*.in; do
  filename="$(basename "${template}")"
  output_path="${OUTPUT_DIR}/${filename%.in}"
  sed \
    -e "s|@REPO_ROOT@|${REPO_ROOT_ESC}|g" \
    -e "s|@ENV_FILE@|${ENV_FILE_ESC}|g" \
    -e "s|@APP_ENV_FILE@|${APP_ENV_FILE_ESC}|g" \
    -e "s|@NETWORK_NAME@|${NETWORK_NAME_ESC}|g" \
    "${template}" > "${output_path}"
  echo "Installed ${output_path}"
done

# systemd prefers unit files from ~/.config/systemd/user over quadlet-generated targets.
# Make the repo authoritative by also installing stageflow.target there.
USER_SYSTEMD_DIR="${HOME}/.config/systemd/user"
USER_TARGET="${USER_SYSTEMD_DIR}/stageflow.target"
RENDERED_TARGET="${OUTPUT_DIR}/stageflow.target"

if [[ ! -f "${RENDERED_TARGET}" ]]; then
  echo "quadlet-install: missing rendered target: ${RENDERED_TARGET}" >&2
  exit 1
fi

mkdir -p "${USER_SYSTEMD_DIR}"

timestamp="$(date +%Y%m%d_%H%M%S)"
backup_dir="${USER_SYSTEMD_DIR}/_backup_stageflow_${timestamp}"
mkdir -p "${backup_dir}"

if [[ -f "${USER_TARGET}" ]]; then
  cp "${USER_TARGET}" "${backup_dir}/stageflow.target"
  echo "Backed up ${USER_TARGET} -> ${backup_dir}/stageflow.target"
fi

cp "${RENDERED_TARGET}" "${USER_TARGET}"
echo "Installed ${USER_TARGET}"

if command -v systemctl >/dev/null 2>&1; then
  systemctl --user daemon-reload
fi
