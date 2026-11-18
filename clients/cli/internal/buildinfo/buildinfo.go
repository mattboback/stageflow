// Package buildinfo holds the version identifiers stamped into release
// binaries via -ldflags (see devtools/scripts/build-cli-release.sh and
// devtools/scripts/install-cli.sh).
package buildinfo

import (
	"runtime/debug"
	"strings"
)

var (
	// Populated at build time via -ldflags.
	Version = "dev"
	Commit  = ""
	Date    = ""
)

// readBuildInfo is debug.ReadBuildInfo, swappable in tests.
var readBuildInfo = debug.ReadBuildInfo

// Resolved returns the version, commit and date to report. Release builds stamp
// all three via -ldflags. `go install …/cmd/stageflow@vX.Y.Z` stamps nothing, so
// the version falls back to the module version the Go toolchain records in the
// binary, and the commit to the build's VCS revision when there is one.
func Resolved() (version, commit, date string) {
	version = strings.TrimSpace(Version)
	commit = strings.TrimSpace(Commit)
	date = strings.TrimSpace(Date)

	if version != "" && version != "dev" {
		return version, commit, date
	}

	version = "dev"

	info, ok := readBuildInfo()
	if !ok {
		return version, commit, date
	}

	if mainVersion := info.Main.Version; mainVersion != "" && mainVersion != "(devel)" {
		version = mainVersion
	}

	if commit == "" {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				commit = setting.Value
			}
		}
	}

	return version, commit, date
}

// FormatVersion renders "version [commit] [date]".
func FormatVersion() string {
	v, c, d := Resolved()
	if len(c) > 12 {
		c = c[:12]
	}

	parts := []string{v}
	if c != "" {
		parts = append(parts, c)
	}

	if d != "" {
		parts = append(parts, d)
	}

	return strings.Join(parts, " ")
}
