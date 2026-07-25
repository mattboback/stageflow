#!/usr/bin/env python3
"""Compares measured Go coverage against a checked-in baseline.

Split out of coverage.sh so the decision logic — which is the part that can be
wrong in a way nobody notices until it waves through a real regression — is
directly testable. See devtools/scripts/tests/coverage-ratchet.test.sh.

Usage:
    coverage-ratchet.py BASELINE MEASURED ALLOWED_DROP_POINTS

Exits 0 when coverage is holding, 1 on a regression or an unreadable input.
"""

from __future__ import annotations

import json
import sys


def load(path: str) -> dict[str, float]:
    with open(path, encoding="utf-8") as handle:
        data = json.load(handle)

    if not isinstance(data, dict):
        raise ValueError(f"{path}: expected a JSON object of module -> percent")

    return {str(module): float(percent) for module, percent in data.items()}


def compare(
    baseline: dict[str, float], measured: dict[str, float], allowed_drop: float
) -> tuple[list[str], list[str]]:
    """Returns (regressions, new_modules) as display-ready strings."""
    regressions: list[str] = []

    for module, before in sorted(baseline.items()):
        if module not in measured:
            # A module that lost all of its tests reports nothing rather than 0%,
            # so absence has to be treated as a regression or deleting a test file
            # would silently pass.
            regressions.append(f"  {module}: was {before:.1f}%, now reports no coverage")
            continue

        now = measured[module]
        if before - now > allowed_drop:
            regressions.append(
                f"  {module}: {before:.1f}% -> {now:.1f}% (-{before - now:.1f} points)"
            )

    return regressions, sorted(set(measured) - set(baseline))


def main(argv: list[str]) -> int:
    if len(argv) != 4:
        print(__doc__, file=sys.stderr)
        return 1

    baseline_path, measured_path, allowed_drop_raw = argv[1:]

    try:
        baseline = load(baseline_path)
        measured = load(measured_path)
        allowed_drop = float(allowed_drop_raw)
    except (OSError, ValueError, json.JSONDecodeError) as error:
        print(f"coverage ratchet: {error}", file=sys.stderr)
        return 1

    regressions, new_modules = compare(baseline, measured, allowed_drop)

    print()
    if new_modules:
        print("New modules, not yet in the baseline:")
        for module in new_modules:
            print(f"  {module}: {measured[module]:.1f}%")
        print("Record them with: just coverage-update")
        print()

    if regressions:
        print(f"Coverage regressed by more than {allowed_drop} points:")
        print("\n".join(regressions))
        print()
        print("Add tests, or accept the change with: just coverage-update")
        return 1

    if measured:
        percents = measured.values()
        print(
            f"Coverage holding across {len(measured)} modules "
            f"(lowest {min(percents):.1f}%, highest {max(percents):.1f}%)."
        )

    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
