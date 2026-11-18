package podman

import (
	"testing"

	"github.com/mattboback/stageflow/libs/go/scannercatalog"
)

// jobMemoryBudgetMB is what one job may claim at once: every scanner and the
// extractor, which serves ZIP sites until the job ends. ClaimNext runs one job
// at a time.
const jobMemoryBudgetMB = 10 * 1024

func TestJobMemoryFitsBudget(t *testing.T) {
	total := extractorMemoryMB
	for _, scanner := range scannercatalog.All() {
		total += scanner.MaxMemoryMB
	}

	if total > jobMemoryBudgetMB {
		t.Fatalf("one job may use %d MB, over the %d MB budget", total, jobMemoryBudgetMB)
	}
}
