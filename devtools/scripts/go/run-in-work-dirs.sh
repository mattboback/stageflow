#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -lt 2 ]]; then
	echo "usage: $0 <label> <command> [args...]" >&2
	exit 2
fi

label="$1"
shift

while IFS= read -r dir; do
	[[ -n "$dir" ]] || continue
	echo "==> ${label} ${dir}"
	(
		cd "$dir"
		"$@"
	)
done < <("$(dirname "$0")/go-work-dirs.sh")
