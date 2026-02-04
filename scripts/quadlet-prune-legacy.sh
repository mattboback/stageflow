#!/usr/bin/env bash
set -euo pipefail

timestamp="$(date +%Y%m%d_%H%M%S)"

containers_dir="${HOME}/.config/containers/systemd"
user_systemd_dir="${HOME}/.config/systemd/user"

backup_containers_dir="${containers_dir}/_backup_stageflow_${timestamp}"
backup_user_systemd_dir="${user_systemd_dir}/_backup_stageflow_${timestamp}"

mkdir -p "${backup_containers_dir}"
mkdir -p "${backup_user_systemd_dir}"

log() {
  echo "stageflow-prune-legacy: $*"
}

move_if_exists() {
  local src="$1"
  local dst_dir="$2"

  if [[ -f "${src}" ]]; then
    mv "${src}" "${dst_dir}/"
    log "Moved $(basename "${src}") -> ${dst_dir}/"
  fi
}

move_container_if_exists() {
  local src="$1"
  local dst_dir="$2"

  if [[ -f "${src}" ]]; then
    local base
    base="$(basename "${src}")"
    mv "${src}" "${dst_dir}/${base}.bak"
    log "Moved ${base} -> ${dst_dir}/${base}.bak"
  fi
}

unit_exists() {
  local unit="$1"
  if command -v rg >/dev/null 2>&1; then
    systemctl --user list-unit-files --no-pager --no-legend 2>/dev/null | awk '{print $1}' | rg -q "^${unit}$"
    return 0
  fi

  systemctl --user list-unit-files --no-pager --no-legend 2>/dev/null | awk '{print $1}' | grep -Eq "^${unit}$"
}

stop_if_active() {
  local unit="$1"
  local state
  state="$(systemctl --user is-active "${unit}" 2>/dev/null || echo "unknown")"
  if [[ "${state}" == "active" ]]; then
    log "Stopping ${unit}..."
    systemctl --user stop "${unit}"
  fi
}

disable_if_enabled() {
  local unit="$1"
  local enabled
  enabled="$(systemctl --user is-enabled "${unit}" 2>/dev/null || echo "unknown")"
  if [[ "${enabled}" == "enabled" ]]; then
    log "Disabling ${unit}..."
    systemctl --user disable "${unit}"
  fi
}

legacy_units=(
  stageflow-portfolio-gateway.service
  stageflow-portfolio-frontend.service
  stageflow-caddy.service
)

for unit in "${legacy_units[@]}"; do
  if unit_exists "${unit}"; then
    stop_if_active "${unit}"
    disable_if_enabled "${unit}"
  fi
done

move_container_if_exists "${containers_dir}/stageflow-portfolio-gateway.container" "${backup_containers_dir}"
move_container_if_exists "${containers_dir}/stageflow-portfolio-frontend.container" "${backup_containers_dir}"
move_container_if_exists "${containers_dir}/stageflow-caddy.container" "${backup_containers_dir}"

# podman-user-generator scans recursively; ensure backups won't be treated as active units.
find "${backup_containers_dir}" -maxdepth 1 -type f -name '*.container' -exec bash -c 'mv "$1" "$1.bak"' _ {} \;

user_target="${user_systemd_dir}/stageflow.target"
if [[ -f "${user_target}" ]]; then
  cp "${user_target}" "${backup_user_systemd_dir}/stageflow.target"
  log "Backed up stageflow.target -> ${backup_user_systemd_dir}/stageflow.target"
fi

app_env="${containers_dir}/stageflow.app.env"
if [[ -f "${app_env}" ]]; then
  if command -v rg >/dev/null 2>&1 && rg -n --fixed-string "stageflow.app.env" "${containers_dir}" "${user_systemd_dir}" >/dev/null 2>&1; then
    log "Keeping stageflow.app.env (still referenced by a unit file)."
  else
    move_if_exists "${app_env}" "${backup_containers_dir}"
  fi
fi

log "Reloading user systemd..."
systemctl --user daemon-reload

log "Done. Backups:"
log "  ${backup_containers_dir}"
log "  ${backup_user_systemd_dir}"
