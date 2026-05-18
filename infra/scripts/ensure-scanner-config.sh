#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
config_file="${SCANNER_CONFIG_FILE:-$repo_root/infra/scanners/scanners.yaml}"
example_file="$repo_root/infra/scanners/scanners.example.yaml"

if [[ -d "$config_file" ]]; then
	if rmdir "$config_file" 2>/dev/null; then
		echo "==> Replaced empty scanner config directory with a file placeholder."
	else
		echo "Scanner config path is a non-empty directory: $config_file" >&2
		echo "Move it aside or replace it with a YAML file before starting the stack." >&2
		exit 1
	fi
fi

if [[ -e "$config_file" && ! -f "$config_file" ]]; then
	echo "Scanner config path exists but is not a regular file: $config_file" >&2
	exit 1
fi

if [[ ! -f "$config_file" ]]; then
	cp "$example_file" "$config_file"
	echo "==> Created local scanner config: $config_file"
fi
