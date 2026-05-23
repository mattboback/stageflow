package jobs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

type fakeAuthUploader struct {
	uploads []fakeUpload
	key     string
	err     error
}

type fakeUpload struct {
	jobID   string
	content []byte
}

func (f *fakeAuthUploader) UploadAuthStorageState(_ context.Context, jobID string, content []byte) (string, error) {
	if f.err != nil {
		return "", f.err
	}

	dup := append([]byte(nil), content...)
	f.uploads = append(f.uploads, fakeUpload{jobID: jobID, content: dup})

	if f.key == "" {
		return jobID + "/auth/storage-state.json", nil
	}

	return f.key, nil
}

func newServiceWithUploader(uploader AuthUploader) *Service {
	s := NewService(nil, nil, nil, nil)
	s.authUploader = uploader

	return s
}

func TestNormalizeJobAuth_FormPassThrough(t *testing.T) {
	t.Parallel()

	s := newServiceWithUploader(&fakeAuthUploader{})

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

	if !strings.Contains(string(out), `"from_env":"STAGEFLOW_AUTH_USER"`) {
		t.Errorf("from_env reference must survive normalization: %s", out)
	}
}

func TestNormalizeJobAuth_StorageStateInlineUploadsAndRewrites(t *testing.T) {
	t.Parallel()

	body := `{"cookies":[],"origins":[]}`
	enc := base64.StdEncoding.EncodeToString([]byte(body))

	uploader := &fakeAuthUploader{key: "job-up/auth/storage-state.json"}
	s := newServiceWithUploader(uploader)

	payload := &events.JobCreatedPayload{
		JobID:     "job-up",
		InputType: "urls",
		URLs:      []string{"https://x"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
			Auth:    json.RawMessage(`{"mode":"storage_state","content_b64":"` + enc + `"}`),
		},
	}

	out, err := s.normalizeJobAuth(context.Background(), payload)
	if err != nil {
		t.Fatalf("normalizeJobAuth: %v", err)
	}

	if len(uploader.uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(uploader.uploads))
	}

	if string(uploader.uploads[0].content) != body {
		t.Errorf("uploaded body mismatch; got=%q want=%q", uploader.uploads[0].content, body)
	}

	wire := string(out)

	if strings.Contains(wire, "content_b64") {
		t.Errorf("normalized auth must not retain content_b64; got %s", wire)
	}

	if !strings.Contains(wire, `"artifact_key":"job-up/auth/storage-state.json"`) {
		t.Errorf("normalized auth missing artifact_key; got %s", wire)
	}
}

func TestNormalizeJobAuth_StorageStateInlineFailsWithoutUploader(t *testing.T) {
	t.Parallel()

	s := newServiceWithUploader(nil)

	body := `{"cookies":[]}`
	enc := base64.StdEncoding.EncodeToString([]byte(body))

	payload := &events.JobCreatedPayload{
		JobID:     "job-noup",
		InputType: "urls",
		URLs:      []string{"https://x"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
			Auth:    json.RawMessage(`{"mode":"storage_state","content_b64":"` + enc + `"}`),
		},
	}

	_, err := s.normalizeJobAuth(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error when AuthUploader is nil")
	}

	if !strings.Contains(err.Error(), "AuthUploader") {
		t.Errorf("expected AuthUploader-mention error, got: %v", err)
	}
}

func TestNormalizeJobAuth_PropagatesUploaderFailure(t *testing.T) {
	t.Parallel()

	s := newServiceWithUploader(&fakeAuthUploader{err: errors.New("minio down")})

	body := `{"cookies":[]}`
	enc := base64.StdEncoding.EncodeToString([]byte(body))

	payload := &events.JobCreatedPayload{
		JobID:     "job-fail",
		InputType: "urls",
		URLs:      []string{"https://x"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
			Auth:    json.RawMessage(`{"mode":"storage_state","content_b64":"` + enc + `"}`),
		},
	}

	_, err := s.normalizeJobAuth(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error from failing uploader")
	}

	if !strings.Contains(err.Error(), "minio down") {
		t.Errorf("error should propagate underlying cause; got %v", err)
	}
}

func TestNormalizeJobAuth_AbsentAuthIsByteIdenticalToPreAuth(t *testing.T) {
	t.Parallel()

	s := newServiceWithUploader(&fakeAuthUploader{})

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
