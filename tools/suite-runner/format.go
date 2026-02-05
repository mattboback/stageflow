package main

import (
	"fmt"
	"io"
)

func printSummary(outcomes []jobOutcome, th Thresholds, out io.Writer) {
	_, _ = fmt.Fprintln(out, "Stageflow Suite Results")
	_, _ = fmt.Fprintln(out, "domain\tstate\tviolations(total/critical/serious)\tjob_id\tpassed")

	for _, o := range outcomes {
		_, _ = fmt.Fprintf(
			out,
			"%s\t%s\t%d/%d/%d\t%s\t%v",
			o.Domain,
			o.State,
			o.TotalViolations,
			o.Critical,
			o.Serious,
			o.JobID,
			o.Passed,
		)

		if o.Error != "" {
			_, _ = fmt.Fprintf(out, "\t(%s)", o.Error)
		}

		_, _ = fmt.Fprintln(out)
	}

	_, _ = fmt.Fprintln(out, "\nThresholds:")
	_, _ = fmt.Fprintf(out, "  max_critical: %v\n", valOr(th.MaxCritical, "-"))
	_, _ = fmt.Fprintf(out, "  max_serious: %v\n", valOr(th.MaxSerious, "-"))
	_, _ = fmt.Fprintf(out, "  max_total:   %v\n", valOr(th.MaxTotal, "-"))
}

func valOr[T any](ptr *T, fallback string) string {
	if ptr == nil {
		return fallback
	}

	return fmt.Sprint(*ptr)
}
