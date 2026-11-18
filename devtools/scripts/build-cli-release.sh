#!/usr/bin/env bash
# Builds the StageFlow CLI release assets for one version: an archive per
# platform plus checksums.txt. The release workflow and `just cli-release` both
# use it, so a release published by hand matches one built by CI.
#
# Usage: build-cli-release.sh vX.Y.Z [out_dir]
# HEAD must be the commit the tag vX.Y.Z points at, with no tracked changes.
# Requires Go, Python 3.9+, Git, tar, zip and sha256sum on PATH.
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
version="${1:-}"
out_dir="${2:-${root_dir}/artifacts/cli-release/${version}}"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "usage: $0 vX.Y.Z [out_dir]" >&2
	exit 2
fi

cd "$root_dir"

for tool in go python3 git tar zip sha256sum; do
	if ! command -v "$tool" >/dev/null; then
		echo "build-cli-release: missing prerequisite: ${tool}" >&2
		exit 2
	fi
done

if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
	echo "build-cli-release: refusing: the checkout has tracked changes" >&2
	exit 1
fi

commit="$(git rev-parse HEAD)"
tag_commit="$(git rev-parse --verify --quiet "refs/tags/${version}^{commit}" || true)"
if [[ "$tag_commit" != "$commit" ]]; then
	echo "build-cli-release: refusing: tag ${version} must point at HEAD (${commit:0:12})" >&2
	exit 1
fi

# The tagged commit's time, so rebuilding the same tag stamps the same date.
build_date="$(TZ=UTC git show -s --format=%cd --date=format-local:%Y-%m-%dT%H:%M:%SZ HEAD)"
buildinfo="github.com/mattboback/stageflow/clients/cli/internal/buildinfo"
ldflags="-s -w -X ${buildinfo}.Version=${version} -X ${buildinfo}.Commit=${commit} -X ${buildinfo}.Date=${build_date}"

rm -rf "$out_dir"
mkdir -p "$out_dir"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

targets=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64)
python3 "${root_dir}/devtools/scripts/cli-third-party-notices.py" "${work_dir}/THIRD_PARTY_NOTICES" "${targets[@]}"

for target in "${targets[@]}"; do
	goos="${target%/*}"
	goarch="${target#*/}"
	bin="stageflow"
	[[ "$goos" == "windows" ]] && bin="stageflow.exe"
	asset="stageflow_${version}_${goos}_${goarch}"

	echo "==> ${asset}"
	rm -rf "${work_dir:?}/${asset}"
	mkdir -p "${work_dir}/${asset}"
	cp "${root_dir}/LICENSE" "${work_dir}/${asset}/LICENSE"
	cp "${work_dir}/THIRD_PARTY_NOTICES" "${work_dir}/${asset}/THIRD_PARTY_NOTICES"
	TZ=UTC CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
		go build -trimpath -ldflags "$ldflags" -o "${work_dir}/${asset}/${bin}" ./clients/cli/cmd/stageflow

	if [[ "$goos" == "windows" ]]; then
		(cd "${work_dir}/${asset}" && zip -q -9 "${out_dir}/${asset}.zip" "$bin" LICENSE THIRD_PARTY_NOTICES)
	else
		tar -C "${work_dir}/${asset}" -czf "${out_dir}/${asset}.tar.gz" "$bin" LICENSE THIRD_PARTY_NOTICES
	fi
done

(cd "$out_dir" && sha256sum stageflow_* >checksums.txt)
echo "==> Assets in ${out_dir#"$root_dir"/}:"
ls -1 "$out_dir"
