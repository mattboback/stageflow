#!/usr/bin/env bash
# Exercise real archive/checksum tooling with a private Git repository and stub
# Go metadata/build output. Real cross-compilation belongs to release validation.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../../.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT
fixture="${test_dir}/repository"
fake_bin="${test_dir}/bin"
export CLI_RELEASE_FIXTURE_DATA="${test_dir}/data"
mkdir -p "${fixture}/devtools/scripts" "$fake_bin" "$CLI_RELEASE_FIXTURE_DATA"
cp "${repo_root}/devtools/scripts/build-cli-release.sh" "${fixture}/devtools/scripts/"
cp "${repo_root}/devtools/scripts/cli-third-party-notices.py" "${fixture}/devtools/scripts/"
cp "${repo_root}/LICENSE" "${fixture}/LICENSE"

python3 - <<'PY'
import os
from pathlib import Path

root = Path(os.environ["CLI_RELEASE_FIXTURE_DATA"])
for directory, files in {
    "goroot": {"LICENSE": "Go fixture license\n", "PATENTS": "Go fixture patent notice\n"},
    "goroot/src/vendor/golang.org/x/fixture": {"LICENSE": "Bundled fixture license\n"},
    "shared": {"LICENSE.md": "Shared fixture license\n", "NOTICE": "Shared fixture attribution\n", "COPYING.LESSER": "Shared supplementary license\n"},
    "windows": {"Licence.txt": "Windows fixture license\n", "PATENTS": "Windows fixture patent notice\n"},
    "fork": {"LICENSE-MIT": "Replacement fixture license without final newline"},
    "unused": {"LICENSE": "Unused fixture license\n"},
}.items():
    target = root / directory
    target.mkdir(parents=True)
    for name, content in files.items():
        (target / name).write_text(content)
(root / "goroot/src/vendor/golang.org/x/fixture/nested").mkdir()
PY

cat >"${fake_bin}/go" <<'PY'
#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys

root = Path(os.environ["CLI_RELEASE_FIXTURE_DATA"])
args = sys.argv[1:]
if args == ["env", "-json", "GOROOT", "GOVERSION"]:
    print(json.dumps({"GOROOT": str(root / "goroot"), "GOVERSION": "go1.26.9"}))
elif args == ["list", "-deps", "-json", "./clients/cli/cmd/stageflow"]:
    assert os.environ["CGO_ENABLED"] == "0"
    packages = [
        {"ImportPath": "example.com/shared", "Module": {"Path": "example.com/shared", "Version": "v1.2.3", "Dir": str(root / "shared")}},
        {"ImportPath": "example.com/shared/nested", "Module": {"Path": "example.com/shared", "Version": "v1.2.3", "Dir": str(root / "shared")}},
        {"ImportPath": "example.com/original", "Module": {"Path": "example.com/original", "Version": "v1.0.0", "Replace": {"Path": "example.com/fork", "Version": "v1.2.0", "Dir": str(root / "fork")}}},
        {"ImportPath": "vendor/golang.org/x/fixture/nested", "Standard": True, "Dir": str(root / "goroot/src/vendor/golang.org/x/fixture/nested")},
        {"ImportPath": "github.com/mattboback/stageflow", "Module": {"Path": "github.com/mattboback/stageflow", "Main": True}},
    ]
    if os.environ["GOOS"] == "windows":
        packages.append({"ImportPath": "example.com/windows", "Module": {"Path": "example.com/windows", "Version": "v2.0.0", "Dir": str(root / "windows")}})
    if os.environ.get("CLI_RELEASE_FIXTURE_REVERSE"):
        packages.reverse()
    for package in packages:
        print(json.dumps(package))
elif args[0] == "build":
    assert os.environ["CGO_ENABLED"] == "0"
    assert "-trimpath" in args and "-ldflags" in args
    output = Path(args[args.index("-o") + 1])
    output.write_text(f"CLI fixture for {os.environ['GOOS']}/{os.environ['GOARCH']}\n")
else:
    sys.exit(f"unexpected go invocation: {args}")
PY
chmod +x "${fake_bin}/go"
export PATH="${fake_bin}:${PATH}"

git -C "$fixture" init -q
git -C "$fixture" add .
git -C "$fixture" -c user.name='CLI release test' -c user.email='cli-release-test@example.invalid' commit -qm 'Private release fixture'
git -C "$fixture" tag v9.9.9
builder="${fixture}/devtools/scripts/build-cli-release.sh"
out_dir="${test_dir}/assets"

expect_refusal() {
	local expected="$1"
	local message="$2"
	shift 2
	local actual=0
	bash "$builder" "$@" >"${test_dir}/refusal.log" 2>&1 || actual=$?
	if [[ "$actual" -ne "$expected" ]] || ! grep -Fq "$message" "${test_dir}/refusal.log"; then
		cat "${test_dir}/refusal.log" >&2
		echo "FAIL: expected exit ${expected} and ${message}, got ${actual}" >&2
		exit 1
	fi
}

expect_refusal 2 'usage:' invalid "$out_dir"
expect_refusal 1 'tag v9.9.8 must point at HEAD' v9.9.8 "$out_dir"
printf '\ntracked change\n' >>"${fixture}/LICENSE"
expect_refusal 1 'the checkout has tracked changes' v9.9.9 "$out_dir"
git -C "$fixture" checkout -- LICENSE

# Verify the new prerequisite is diagnosed before trying to produce assets.
no_python="${test_dir}/no-python"
mkdir -p "$no_python"
for tool in dirname go git tar zip sha256sum; do
	ln -s "$(command -v "$tool")" "${no_python}/${tool}"
done
actual=0
PATH="$no_python" /bin/bash "$builder" v9.9.9 "$out_dir" >"${test_dir}/prerequisite.log" 2>&1 || actual=$?
if [[ "$actual" -ne 2 ]] || ! grep -Fq 'missing prerequisite: python3' "${test_dir}/prerequisite.log"; then
	cat "${test_dir}/prerequisite.log" >&2
	exit 1
fi

# Changing dependency traversal or target order must not change notice bytes.
targets=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64)
helper="${fixture}/devtools/scripts/cli-third-party-notices.py"
python3 "$helper" "${test_dir}/notices-first" "${targets[@]}"
CLI_RELEASE_FIXTURE_REVERSE=1 python3 "$helper" "${test_dir}/notices-second" windows/amd64 darwin/arm64 darwin/amd64 linux/arm64 linux/amd64
cmp "${test_dir}/notices-first" "${test_dir}/notices-second"

CGO_ENABLED=1 bash "$builder" v9.9.9 "$out_dir" >"${test_dir}/build.log"
python3 - "$out_dir" "$repo_root" "${test_dir}/notices-first" <<'PY'
import hashlib
from pathlib import Path
import sys
import tarfile
import zipfile

assets, repo, notices_file = map(Path, sys.argv[1:])
expected_notices = notices_file.read_bytes()
expected_license = (repo / "LICENSE").read_bytes()
archives = sorted(assets.glob("stageflow_*"))
assert len(archives) == 5, archives
checksums = dict(line.split()[::-1] for line in (assets / "checksums.txt").read_text().splitlines())
assert set(checksums) == {archive.name for archive in archives}, checksums
for archive in archives:
    windows = archive.suffix == ".zip"
    binary = "stageflow.exe" if windows else "stageflow"
    if windows:
        with zipfile.ZipFile(archive) as contents:
            assert sorted(contents.namelist()) == sorted([binary, "LICENSE", "THIRD_PARTY_NOTICES"])
            license_text, notices = contents.read("LICENSE"), contents.read("THIRD_PARTY_NOTICES")
    else:
        with tarfile.open(archive) as contents:
            assert sorted(contents.getnames()) == sorted([binary, "LICENSE", "THIRD_PARTY_NOTICES"])
            license_text = contents.extractfile("LICENSE").read()
            notices = contents.extractfile("THIRD_PARTY_NOTICES").read()
    assert license_text == expected_license
    assert notices == expected_notices
    assert checksums[archive.name] == hashlib.sha256(archive.read_bytes()).hexdigest()
for text in [
    b"Go runtime and standard library (go1.26.9)", b"Go fixture license\n", b"Go fixture patent notice\n",
    b"Bundled fixture license\n", b"Shared fixture license\n", b"Shared fixture attribution\n",
    b"Shared supplementary license\n", b"Windows fixture license\n", b"Windows fixture patent notice\n",
    b"Replacement fixture license without final newline\n", b"example.com/shared v1.2.3",
    b"example.com/windows v2.0.0\nTargets: windows/amd64\n", b"replaced by example.com/fork v1.2.0",
]:
    assert text in expected_notices, text
assert expected_notices.count(b"example.com/shared v1.2.3") == 1
assert b"Unused fixture license" not in expected_notices
assert str(repo).encode() not in expected_notices
print("cli-release archive/notice/checksum tests passed")
PY

# Missing/empty licenses must stop release packaging rather than silently omit
# dependency obligations or produce a partially licensed archive set.
mv "${CLI_RELEASE_FIXTURE_DATA}/windows/Licence.txt" "${test_dir}/saved-license"
expect_refusal 1 'no license text found for example.com/windows' v9.9.9 "$out_dir"
mv "${test_dir}/saved-license" "${CLI_RELEASE_FIXTURE_DATA}/windows/Licence.txt"
: >"${CLI_RELEASE_FIXTURE_DATA}/windows/Licence.txt"
expect_refusal 1 'empty notice text for example.com/windows' v9.9.9 "$out_dir"
echo 'cli-release refusal tests passed'
