package api

import "testing"

func TestJobScopedKey(t *testing.T) {
	t.Parallel()

	jobID := "job-123"

	if key, ok := jobScopedKey(jobID, "job-123/axe/page-1/results.json"); !ok || key == "" {
		t.Fatalf("expected key to be accepted, got ok=%v key=%q", ok, key)
	}

	if _, ok := jobScopedKey(jobID, "job-123/../../evil"); ok {
		t.Fatalf("expected escaping key to be rejected")
	}

	if _, ok := jobScopedKey(jobID, "../../evil"); ok {
		t.Fatalf("expected non-scoped key to be rejected")
	}
}

func TestJobScopedJoin(t *testing.T) {
	t.Parallel()

	jobID := "job-123"

	if key, ok := jobScopedJoin(jobID, "axe", "page-1", "screenshots", "shot.png"); !ok || key == "" {
		t.Fatalf("expected joined key to be accepted, got ok=%v key=%q", ok, key)
	}

	// Two .. segments only escape scanner/page, but remain within job.
	if _, ok := jobScopedJoin(jobID, "axe", "page-1", "../../safe-within-job"); !ok {
		t.Fatalf("expected join to remain job-scoped")
	}

	// Three .. segments escapes the job prefix and must be rejected.
	if _, ok := jobScopedJoin(jobID, "axe", "page-1", "../../../evil"); ok {
		t.Fatalf("expected join to reject job prefix escape")
	}
}
