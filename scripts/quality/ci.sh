#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}/verify-entrypoint-parity.sh"
"${SCRIPT_DIR}/lint.sh"
"${SCRIPT_DIR}/test.sh"
