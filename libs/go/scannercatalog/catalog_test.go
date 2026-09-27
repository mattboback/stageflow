package scannercatalog

import (
	"sort"
	"testing"
)

func TestBuiltinManifests(t *testing.T) {
	manifests, err := BuiltinManifests()
	if err != nil {
		t.Fatalf("BuiltinManifests() error: %v", err)
	}

	if len(manifests) == 0 {
		t.Fatal("BuiltinManifests() returned no manifests")
	}

	gotIDs := make([]string, 0, len(manifests))
	for _, m := range manifests {
		gotIDs = append(gotIDs, m.Id)
	}

	sort.Strings(gotIDs)

	want := []string{
		"axe",
		"lighthouse",
		"link-checker",
		"security-headers",
		"seo",
	}

	if len(gotIDs) != len(want) {
		t.Fatalf("builtin manifest ids = %v, want %v", gotIDs, want)
	}

	for i := range want {
		if gotIDs[i] != want[i] {
			t.Fatalf("builtin manifest ids = %v, want %v", gotIDs, want)
		}
	}
}

// maxConcurrentJobs is the job concurrency the memory budget allows for. PR 2.3
// adds the job cap and starts it at this value.
const maxConcurrentJobs = 1

// jobMemoryBudgetMB is what concurrent jobs may claim together. A job runs every
// scanner at once, so a job claims the sum of the manifests' limits.
const jobMemoryBudgetMB = 10 * 1024

func TestManifestMemoryLimitsFitBudget(t *testing.T) {
	manifests, err := BuiltinManifests()
	if err != nil {
		t.Fatalf("BuiltinManifests() error: %v", err)
	}

	perJobMB := 0

	for _, m := range manifests {
		if m.Requirements == nil || m.Requirements.Resources == nil || m.Requirements.Resources.MaxMemoryMB == nil {
			t.Fatalf("%s: manifest must set requirements.resources.maxMemoryMB", m.Id)
		}

		perJobMB += *m.Requirements.Resources.MaxMemoryMB
	}

	if total := perJobMB * maxConcurrentJobs; total > jobMemoryBudgetMB {
		t.Fatalf("scanner memory limits = %d MB per job x %d jobs = %d MB, over the %d MB budget",
			perJobMB, maxConcurrentJobs, total, jobMemoryBudgetMB)
	}
}
