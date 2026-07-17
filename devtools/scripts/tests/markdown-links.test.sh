#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
checker="$repo_root/devtools/scripts/check-markdown-links.mjs"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT

git -C "$fixture" init -q
mkdir -p "$fixture/docs"
touch "$fixture/docs/target.md"
cat >"$fixture/.gitignore" <<'EOF'
docs/generated/
EOF
cat >"$fixture/docs/good.md" <<'EOF'
[Target](target.md)
[Section](target.md#heading)
[External](https://example.com)
`docs/target.md`
`docs/generated`
EOF

(cd "$fixture" && node "$checker" docs/good.md) >/dev/null

cat >"$fixture/docs/bad.md" <<'EOF'
[Missing](not-there.md)
`docs/also-missing.md`
EOF

if (cd "$fixture" && node "$checker" docs/bad.md) >"$fixture/output" 2>&1; then
	echo "expected broken link check to fail" >&2
	exit 1
fi

grep -q 'not-there.md' "$fixture/output"
grep -q 'also-missing.md' "$fixture/output"
echo "markdown-link tests passed"
