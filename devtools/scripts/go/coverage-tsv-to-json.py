#!/usr/bin/env python3
"""Converts coverage.sh's `module<TAB>percent` scratch file into stable JSON.

Sorted keys and a trailing newline so the checked-in baseline produces a readable
diff when a module's coverage genuinely moves.

Usage:
    coverage-tsv-to-json.py MEASURED_TSV OUT_JSON
"""

from __future__ import annotations

import json
import sys


def main(argv: list[str]) -> int:
    if len(argv) != 3:
        print(__doc__, file=sys.stderr)
        return 1

    rows: dict[str, float] = {}
    with open(argv[1], encoding="utf-8") as handle:
        for line in handle:
            line = line.rstrip("\n")
            if not line:
                continue
            module, percent = line.split("\t")
            rows[module] = float(percent)

    with open(argv[2], "w", encoding="utf-8") as handle:
        json.dump(rows, handle, indent=2, sort_keys=True)
        handle.write("\n")

    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
