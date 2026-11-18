#!/usr/bin/env python3
"""Collect CLI notices from the release toolchain and target-specific imports."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys


NOTICE_NAME = re.compile(r"^(licen[cs]e|copying|notice|patents)(?:[._-].*)?$", re.I)
LICENSE_NAME = re.compile(r"^(licen[cs]e|copying)(?:[._-].*)?$", re.I)
ROOT = Path(__file__).resolve().parents[2]


def go(*args, env=None):
    return subprocess.check_output(["go", *args], cwd=ROOT, env=env, text=True)


def json_objects(data):
    # go list -json emits consecutive objects rather than a JSON array.
    decoder = json.JSONDecoder()
    while data.strip():
        value, end = decoder.raw_decode(data.lstrip())
        yield value
        data = data.lstrip()[end:]


def notice_files(directory):
    return sorted(
        (path for path in directory.iterdir() if path.is_file() and NOTICE_NAME.fullmatch(path.name)),
        key=lambda path: path.name,
    )


def section(label, directory, targets):
    files = notice_files(directory)
    if not any(LICENSE_NAME.fullmatch(path.name) for path in files):
        raise ValueError(f"no license text found for {label}")
    parts = [f"\n{'=' * 72}\n{label}\nTargets: {', '.join(sorted(targets))}\n".encode()]
    for path in files:
        content = path.read_bytes()
        if not content.strip():
            raise ValueError(f"empty notice text for {label}: {path.name}")
        parts.extend([f"\n--- {path.name} ---\n".encode(), content])
        if not content.endswith(b"\n"):
            parts.append(b"\n")
    return b"".join(parts)


def collect(targets):
    toolchain = json.loads(go("env", "-json", "GOROOT", "GOVERSION"))
    goroot = Path(toolchain["GOROOT"])
    modules = {}
    bundled = {}
    for target in targets:
        goos, goarch = target.split("/")
        env = dict(os.environ, CGO_ENABLED="0", GOOS=goos, GOARCH=goarch)
        # Module metadata is attached to each imported package; do not include
        # unrelated go.mod requirements or test-only imports. Official schema:
        # https://pkg.go.dev/cmd/go#hdr-List_packages_or_modules
        for package in json_objects(go("list", "-deps", "-json", "./clients/cli/cmd/stageflow", env=env)):
            module = package.get("Module")
            if module and not module.get("Main"):
                source = module.get("Replace") or module
                label = f"{module['Path']} {module['Version']}"
                if module.get("Replace"):
                    label += f" (replaced by {source['Path']} {source.get('Version', 'local')})"
                directory = Path(source["Dir"])
                if label in modules and modules[label][0] != directory:
                    raise ValueError(f"conflicting source directories for {label}")
                modules.setdefault(label, (directory, set()))[1].add(target)
            elif package.get("Standard") and package["ImportPath"].startswith("vendor/"):
                # Preserve separate notices for libraries bundled in Go's
                # standard library, using only imports in this CLI's graph.
                directory = Path(package["Dir"])
                vendor = goroot / "src" / "vendor"
                while directory != vendor and directory.is_relative_to(vendor):
                    if any(LICENSE_NAME.fullmatch(path.name) for path in notice_files(directory)):
                        label = f"Go bundled {directory.relative_to(goroot / 'src')} ({toolchain['GOVERSION']})"
                        bundled.setdefault(label, (directory, set()))[1].add(target)
                        break
                    directory = directory.parent
                else:
                    raise ValueError(f"no bundled license found for {package['ImportPath']}")

    parts = [
        b"StageFlow CLI third-party notices\n\n"
        b"License and notice texts from the modules imported by the supported release\n"
        b"targets and the Go toolchain used to build them. The project license is\n"
        b"provided separately in LICENSE.\n"
    ]
    parts.append(section(f"Go runtime and standard library ({toolchain['GOVERSION']})", goroot, targets))
    for label, (directory, applicable_targets) in sorted({**bundled, **modules}.items()):
        parts.append(section(label, directory, applicable_targets))
    return b"".join(parts)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path)
    parser.add_argument("targets", nargs="+", help="release GOOS/GOARCH targets")
    args = parser.parse_args()
    for target in args.targets:
        if not re.fullmatch(r"[a-z0-9]+/[a-z0-9]+", target):
            parser.error(f"invalid release target: {target}")
    try:
        args.output.write_bytes(collect(sorted(set(args.targets))))
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        print(f"cli-third-party-notices: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
