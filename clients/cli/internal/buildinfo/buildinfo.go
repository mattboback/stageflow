// Package buildinfo holds the version identifiers stamped into release
// binaries via -ldflags (see .github/workflows/release-stageflow-cli.yml and
// devtools/scripts/install-cli.sh).
package buildinfo

import "strings"

var (
	// Populated at build time via -ldflags.
	Version = "dev"
	Commit  = ""
	Date    = ""
)

// FormatVersion renders "version [commit] [date]" with a dev fallback.
func FormatVersion() string {
	v := strings.TrimSpace(Version)
	if v == "" {
		v = "dev"
	}

	c := strings.TrimSpace(Commit)
	if len(c) > 12 {
		c = c[:12]
	}

	d := strings.TrimSpace(Date)

	parts := []string{v}
	if c != "" {
		parts = append(parts, c)
	}

	if d != "" {
		parts = append(parts, d)
	}

	return strings.Join(parts, " ")
}
