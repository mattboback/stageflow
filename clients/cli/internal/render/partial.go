package render

import (
	"fmt"
	"io"
	"strings"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/diff"
)

// WarnIfPartial writes a warning to out when scanners in doc did not finish,
// naming each with its error, and returns their IDs. It writes nothing and
// returns nil for a complete report.
func WarnIfPartial(out io.Writer, doc report.UnifiedReportV2) ([]string, error) {
	incomplete := diff.IncompleteScanners(doc.Scanners)
	if len(incomplete) == 0 {
		return nil, nil
	}

	lines := []string{"Warning: partial report. These scanners did not finish, so their findings are missing:"}

	for _, scanner := range doc.Scanners {
		if !containsString(incomplete, scanner.Id) {
			continue
		}

		reason := string(scanner.Status)
		if scanner.Error != nil && *scanner.Error != "" {
			reason = *scanner.Error
		}

		lines = append(lines, fmt.Sprintf("  %s: %s", scanner.Id, reason))
	}

	_, err := fmt.Fprintln(out, strings.Join(lines, "\n"))

	return incomplete, err
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
