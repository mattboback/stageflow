#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CONFIG="$ROOT_DIR/.github/dependabot.yml"

ecosystem_directories() {
	local ecosystem="$1"
	awk -v target="$ecosystem" '
		/^  - package-ecosystem: / {
			active = ($0 == "  - package-ecosystem: \"" target "\"")
			next
		}
		active && /^    directory: "\// {
			line = $0
			sub(/^    directory: "/, "", line)
			sub(/"$/, "", line)
			print line
			next
		}
		active && /^      - "\// {
			line = $0
			sub(/^      - "/, "", line)
			sub(/"$/, "", line)
			print line
		}
	' "$CONFIG"
}

check_ecosystem() {
	local ecosystem="$1"
	local manifest_name="$2"
	local missing=0
	declare -A covered=()

	while IFS= read -r directory; do
		[[ -n "$directory" ]] && covered["$directory"]=1
	done < <(ecosystem_directories "$ecosystem")

	while IFS= read -r manifest; do
		local directory="${manifest%/"$manifest_name"}"
		if [[ "$directory" == "$manifest" || "$directory" == "." ]]; then
			directory="/"
		else
			directory="/${directory#./}"
		fi

		if [[ -z "${covered[$directory]:-}" ]]; then
			printf 'Dependabot %s coverage missing for %s\n' "$ecosystem" "$directory" >&2
			missing=1
		fi
	done < <(cd "$ROOT_DIR" && git ls-files | awk -v name="$manifest_name" '$0 == name || $0 ~ ("/" name "$")')

	return "$missing"
}

status=0
check_ecosystem gomod go.mod || status=1
check_ecosystem bun package.json || status=1

if [[ "$status" -ne 0 ]]; then
	exit "$status"
fi

echo "Dependabot covers every tracked Go and Bun package"
