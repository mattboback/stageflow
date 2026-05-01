#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

bin_dir="${1:-$HOME/.local/bin}"
bin_name="${2:-stageflow}"
dest="${bin_dir}/${bin_name}"

mkdir -p "$bin_dir"

tmp="$(mktemp -t stageflow-cli.XXXXXX)"
trap 'rm -f "$tmp"' EXIT

echo "==> Building StageFlow CLI..."
version="$(cd "${REPO_ROOT}" && git describe --tags --always --dirty 2>/dev/null || echo dev)"
commit="$(cd "${REPO_ROOT}" && git rev-parse --short HEAD 2>/dev/null || echo "")"
build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
ldflags="-s -w -X main.version=${version} -X main.commit=${commit} -X main.date=${build_date}"
(cd "${REPO_ROOT}/clients/cli" && go build -trimpath -ldflags "$ldflags" -o "$tmp" .)

echo "==> Installing: $dest"
install -m 0755 "$tmp" "$dest"

resolved="$(command -v "$bin_name" || true)"
if [[ -z "$resolved" ]]; then
	echo "Installed '$dest', but '$bin_dir' is not on PATH. Add it to PATH, then run '${bin_name} version'." >&2
	exit 1
fi

real_resolved="$(realpath "$resolved" 2>/dev/null || readlink -f "$resolved" 2>/dev/null || echo "$resolved")"
real_dest="$(realpath "$dest" 2>/dev/null || readlink -f "$dest" 2>/dev/null || echo "$dest")"

if [[ "$real_resolved" != "$real_dest" ]]; then
	echo "ERROR: Installed '$dest', but '$bin_name' does not resolve to the installed binary (resolves to '$resolved')." >&2
	echo "You may need to reorder PATH or remove the stale binary." >&2
	exit 1
fi

echo "==> Installed and available on PATH as '$bin_name'."
"$resolved" version
