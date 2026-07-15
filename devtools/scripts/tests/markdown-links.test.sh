#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
checker="$repo_root/devtools/scripts/check-markdown-links.mjs"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT

mkdir -p "$fixture/docs"
touch "$fixture/docs/target.md"
cat >"$fixture/docs/good.md" <<'EOF'
[Target](target.md)
[Section](target.md#heading)
[External](https://example.com)
EOF

node "$checker" "$fixture/docs/good.md" >/dev/null

cat >"$fixture/docs/bad.md" <<'EOF'
[Missing](not-there.md)
EOF

if node "$checker" "$fixture/docs/bad.md" >"$fixture/output" 2>&1; then
	echo "expected broken link check to fail" >&2
	exit 1
fi

grep -q 'not-there.md' "$fixture/output"
echo "markdown-link tests passed"
