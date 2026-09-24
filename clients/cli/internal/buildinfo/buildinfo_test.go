package buildinfo

import (
	"runtime/debug"
	"testing"
)

func withBuild(t *testing.T, version, commit, date string, info *debug.BuildInfo) {
	t.Helper()

	savedVersion, savedCommit, savedDate, savedRead := Version, Commit, Date, readBuildInfo

	t.Cleanup(func() {
		Version, Commit, Date, readBuildInfo = savedVersion, savedCommit, savedDate, savedRead
	})

	Version, Commit, Date = version, commit, date
	readBuildInfo = func() (*debug.BuildInfo, bool) { return info, info != nil }
}

func TestResolvedPrefersLdflagsStamp(t *testing.T) {
	withBuild(t, "v0.4.1", "6d296c877626164073e0cb9a00df9bf4f16d6292", "2026-09-24T20:52:25Z",
		&debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}})

	if got := FormatVersion(); got != "v0.4.1 6d296c877626 2026-09-24T20:52:25Z" {
		t.Fatalf("FormatVersion() = %q", got)
	}
}

func TestResolvedFallsBackToModuleVersionForGoInstall(t *testing.T) {
	withBuild(t, "dev", "", "", &debug.BuildInfo{Main: debug.Module{Version: "v0.4.1"}})

	version, commit, date := Resolved()
	if version != "v0.4.1" || commit != "" || date != "" {
		t.Fatalf("Resolved() = %q, %q, %q", version, commit, date)
	}
}

func TestResolvedUsesVCSRevisionForLocalBuilds(t *testing.T) {
	withBuild(t, "", "", "", &debug.BuildInfo{
		Main:     debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef0123456789"}},
	})

	if got := FormatVersion(); got != "dev abcdef012345" {
		t.Fatalf("FormatVersion() = %q", got)
	}
}

func TestResolvedWithoutBuildInfo(t *testing.T) {
	withBuild(t, "dev", "", "", nil)

	if got := FormatVersion(); got != "dev" {
		t.Fatalf("FormatVersion() = %q", got)
	}
}
