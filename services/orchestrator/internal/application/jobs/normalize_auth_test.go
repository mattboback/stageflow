package jobs

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

type fakeAuthCleaner struct {
	deleted []string
	err     error
}

func (f *fakeAuthCleaner) DeleteAuthStorageState(_ context.Context, key string) error {
	if f.err != nil {
		return f.err
	}

	f.deleted = append(f.deleted, key)

	return nil
}

func TestNormalizeJobAuth_FormPassThrough(t *testing.T) {
	t.Parallel()

	s := NewService(nil, nil, nil, nil)

	payload := &events.JobCreatedPayload{
		JobID:     "job-form",
		InputType: "urls",
		URLs:      []string{"https://x"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
			Auth:    json.RawMessage(authFormFixture),
		},
	}

	out, err := s.normalizeJobAuth(context.Background(), payload)
	if err != nil {
		t.Fatalf("normalizeJobAuth: %v", err)
	}

	if !strings.Contains(string(out), `"value":"demo@example.com"`) {
		t.Errorf("form recipe must survive normalization: %s", out)
	}
}

func TestNormalizeJobAuthRejectsInlineSessionBeforeDispatch(t *testing.T) {
	s := NewService(nil, nil, nil, nil)

	payload := &events.JobCreatedPayload{
		JobID:  "job",
		Config: models.JobConfig{Auth: json.RawMessage(`{"mode":"storage_state","content_b64":"e30="}`)},
	}
	if _, err := s.normalizeJobAuth(context.Background(), payload); err == nil {
		t.Fatal("inline session accepted downstream of intake")
	}

	payload.Config.Auth = json.RawMessage(`{"mode":"storage_state"}`)

	out, err := s.normalizeJobAuth(context.Background(), payload)
	if err != nil || string(out) != `{"mode":"storage_state"}` {
		t.Fatalf("mounted auth not preserved: %s %v", out, err)
	}
}

func TestNormalizeJobAuth_AbsentAuthIsByteIdenticalToPreAuth(t *testing.T) {
	t.Parallel()

	s := NewService(nil, nil, nil, nil)

	payload := &events.JobCreatedPayload{
		JobID:     "job-bare",
		InputType: "urls",
		URLs:      []string{"https://x"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	out, err := s.normalizeJobAuth(context.Background(), payload)
	if err != nil {
		t.Fatalf("normalizeJobAuth: %v", err)
	}

	if out != nil {
		t.Errorf("absent auth must return nil so JobConfig.Auth omitempty kicks in; got %s", out)
	}
}

func TestCleanupAuthStorageStateDeletesStorageStateCredential(t *testing.T) {
	t.Parallel()

	cleaner := &fakeAuthCleaner{}
	s := NewService(nil, nil, nil, nil, WithAuthCleaner(cleaner))

	s.cleanupAuthStorageState(context.Background(), &models.Job{
		ID: "job-clean",
		Config: models.JobConfig{
			Auth: json.RawMessage(`{"mode":"storage_state","artifact_key":"job-clean/auth.json"}`),
		},
	})

	if len(cleaner.deleted) != 1 || cleaner.deleted[0] != "job-clean/auth.json" {
		t.Fatalf("deleted auth keys = %v", cleaner.deleted)
	}
}

func TestCleanupAuthStorageStateIgnoresFormAuth(t *testing.T) {
	t.Parallel()

	cleaner := &fakeAuthCleaner{}
	s := NewService(nil, nil, nil, nil, WithAuthCleaner(cleaner))

	s.cleanupAuthStorageState(context.Background(), &models.Job{
		ID: "job-form",
		Config: models.JobConfig{
			Auth: json.RawMessage(authFormFixture),
		},
	})

	if len(cleaner.deleted) != 0 {
		t.Fatalf("expected no deletes for form auth, got %v", cleaner.deleted)
	}
}
