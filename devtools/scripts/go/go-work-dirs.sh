#!/usr/bin/env bash
set -euo pipefail

if ! command -v go >/dev/null 2>&1; then
  echo "go is required but was not found in PATH" >&2
  exit 1
fi

if [ ! -f go.work ]; then
  echo "go.work not found in the current directory: $(pwd)" >&2
  exit 1
fi

go work edit -json | python3 -c '
import json
import sys

data = json.load(sys.stdin)
for entry in data.get("Use", []):
    disk_path = entry.get("DiskPath")
    if disk_path:
        print(disk_path)
'
