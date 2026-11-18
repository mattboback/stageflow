#!/usr/bin/env python3
"""Check production Go reachability; temporary findings have cleanup owners and reject stale entries."""

import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile


ROOT = Path(__file__).resolve().parents[2]
MODULE = "github.com/mattboback/stageflow/"
MAINS = {
    MODULE + path
    for path in (
        "clients/cli/cmd/stageflow",
        "devtools/cmd/sanitize-backup",
        "services/archive-extractor/cmd/server",
        "services/orchestrator/cmd/orchestrator",
        "services/orchestrator/cmd/healthcheck",
        "services/platform-api/cmd/server",
        "services/platform-api/cmd/healthcheck",
    )
}
CLI = MODULE + "clients/cli/cmd/stageflow"


def run(args, cwd, env):
    return subprocess.run(
        args, cwd=cwd, env=env, check=True, text=True, stdout=subprocess.PIPE
    ).stdout


def findings(output, root):
    # deadcode emits null when no functions are unreachable and exits zero even
    # when it reports findings. Never use its exit status as the policy check.
    result = {}
    for package in json.loads(output) or []:
        for function in package["Funcs"]:
            key = (package["Path"], function["Name"])
            result[key] = (root / function["Position"]["File"]).resolve()
    return result


def compiled_files(output):
    decoder = json.JSONDecoder()
    files = set()
    while output.strip():
        package, end = decoder.raw_decode(output.lstrip())
        output = output.lstrip()[end:]
        for name in package.get("GoFiles", []) + package.get("CgoFiles", []):
            files.add((Path(package["Dir"]) / name).resolve())
    return files


def unreachable_in_every_build(builds):
    candidates = {}
    for dead, _ in builds:
        candidates.update(dead)
    # A shared function live in Windows must not be deleted merely because its
    # callers are absent on Linux. OS-specific files only count where compiled.
    return {
        key
        for key, file in candidates.items()
        if all(file not in files or key in dead for dead, files in builds)
    }


def load_temporary(path):
    entries = json.loads(path.read_text())
    allowed = {}
    for entry in entries:
        key = (entry["package"], entry["function"])
        if (
            key in allowed
            or not isinstance(entry["owner"], str)
            or not entry["owner"]
            or any(character.isspace() for character in entry["owner"])
            or not isinstance(entry["reason"], str)
            or not entry["reason"].strip()
        ):
            raise ValueError(f"invalid or duplicate temporary finding: {key}")
        allowed[key] = entry["owner"]
    return allowed


def differences(actual, allowed):
    errors = [
        f"new unreachable function: {package}.{function}"
        for package, function in sorted(actual - allowed.keys())
    ]
    errors += [
        f"stale temporary finding (remove entry for {allowed[key]}): {key[0]}.{key[1]}"
        for key in sorted(allowed.keys() - actual)
    ]
    return errors


def self_test(tool, env):
    with tempfile.TemporaryDirectory(prefix="stageflow-deadcode-probe-") as temp:
        root = Path(temp)
        (root / "go.mod").write_text("module example.com/deadcode-probe\n\ngo 1.26.0\n")
        (root / "main.go").write_text(
            "package main\nfunc main() { reachable() }\n"
            "func reachable() {}\nfunc unreachable() {}\n"
        )
        actual = findings(run([tool, "-json", "."], root, env), root)
        key = ("example.com/deadcode-probe", "unreachable")
        if set(actual) != {key} or not differences(set(actual), {}):
            raise ValueError("deadcode gate did not detect an unreachable production function")
        if not differences(set(), {key: "reachability-cleanup"}):
            raise ValueError("deadcode gate did not reject a stale temporary finding")
        if differences(set(actual), {key: "reachability-cleanup"}):
            raise ValueError("deadcode gate rejected an exact temporary finding")
        ledger = root / "temporary.json"
        entry = {"package": key[0], "function": key[1], "owner": "reachability-cleanup", "reason": "Remove the isolated probe"}
        ledger.write_text(json.dumps([entry]))
        if load_temporary(ledger) != {key: "reachability-cleanup"}:
            raise ValueError("deadcode gate rejected an exact owned finding")
        for entries in ([entry, entry], [{**entry, "owner": "  "}], [{**entry, "reason": "  "}]):
            ledger.write_text(json.dumps(entries))
            try:
                load_temporary(ledger)
            except ValueError:
                pass
            else:
                raise ValueError("deadcode gate accepted an invalid or duplicate finding")
        files = {root / "main.go"}
        if unreachable_in_every_build([(actual, files), ({}, files)]):
            raise ValueError("deadcode gate classified code live on another platform as dead")
        if findings("null", root):
            raise ValueError("deadcode gate mishandled an empty report")
    print("Go deadcode detection and temporary-finding checks passed")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    go = os.environ.get("GO", "go")
    tool = os.environ.get("DEADCODE", "deadcode")
    linux = {**os.environ, "GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "1"}
    if args.self_test:
        self_test(tool, linux)
        return 0

    mains = set(
        run([go, "list", "-f", '{{if eq .Name "main"}}{{.ImportPath}}{{end}}', "./..."], ROOT, linux).split()
    )
    if mains != MAINS:
        raise ValueError(f"production entrypoints changed; review gate roots: {sorted(mains)}")

    builds = []
    for roots, env in (
        (sorted(mains), linux),
        ([CLI], {**linux, "GOOS": "windows", "CGO_ENABLED": "0"}),
    ):
        files = compiled_files(run([go, "list", "-deps", "-json", *roots], ROOT, env))
        dead = findings(run([tool, "-json", *roots], ROOT, env), ROOT)
        builds.append((dead, files))

    actual = unreachable_in_every_build(builds)
    allowed = load_temporary(ROOT / "devtools/go-deadcode-temporary.json")
    errors = differences(actual, allowed)
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print(f"Go production reachability passed ({len(mains)} entrypoints, {len(allowed)} temporary findings)")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError) as error:
        print(f"Go deadcode check failed: {error}", file=sys.stderr)
        sys.exit(1)
