package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/mattboback/stageflow/internal/archive"
	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/internal/testpostgres"
	"github.com/mattboback/stageflow/libs/go/models"
)

type lostCommitAdmission struct {
	database *jobstore.Database
	delayed  bool
	job      *models.Job
}

func (s *lostCommitAdmission) AdmitJob(ctx context.Context, job *models.Job) error {
	s.job = job
	if !s.delayed {
		if err := s.database.AdmitJob(ctx, job); err != nil {
			return err
		}
	}

	return io.ErrUnexpectedEOF
}
func uncertainTestDatabase(t *testing.T) *jobstore.Database {
	t.Helper()

	address, stop, err := testpostgres.PostgresForTests("uncertain-admission")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if stopErr := stop(); stopErr != nil {
			t.Error(stopErr)
		}
	})

	admin, err := sql.Open("pgx", address)
	if err != nil {
		t.Fatal(err)
	}

	schema := fmt.Sprintf("admission_%d", time.Now().UnixNano())
	if _, schemaErr := admin.Exec("CREATE SCHEMA " + schema); schemaErr != nil {
		t.Fatal(schemaErr)
	}

	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}

	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()

	database, err := jobstore.NewDatabase(&jobstore.Config{URL: parsed.String()})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = database.Close()
		_, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		_ = admin.Close()
	})

	return database
}
func TestUncertainAdmissionPreservesInputsAndLateCommittedJobs(t *testing.T) {
	database := uncertainTestDatabase(t)

	for _, delayed := range []bool{false, true} {
		for _, input := range []string{"urls", "zip"} {
			t.Run(
				fmt.Sprintf("%s/delayed-%t", input, delayed),
				func(t *testing.T) { testUncertainAdmission(t, database, input, delayed) },
			)
		}
	}
}
func testUncertainAdmission(t *testing.T, database *jobstore.Database, input string, delayed bool) {
	t.Helper()

	ctx := context.Background()
	server, _, _ := newTestServer(t)

	files, err := artifacts.NewFiles(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = files.Close() })

	admission := &lostCommitAdmission{database: database, delayed: delayed}
	server.config.Admitter = admission
	server.config.Storage = files
	server.statusReader = jobstore.StatusReader{Database: database}
	server.config.ValidateZIP = func(ctx context.Context, id string) error { return validateUncertainZIP(ctx, files, id) }

	var request *http.Request

	expectedFile := "auth.json"

	zipBytes := buildTestZip(t)
	if input == "zip" {
		request = uploadArchiveRequest(t, zipBytes)
		expectedFile = "input.zip"
	} else {
		request = httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewBufferString(storageStateBody))
	}

	response := httptest.NewRecorder()
	if input == "zip" {
		server.handleJobZipUpload(response, request)
	} else {
		server.handleJobURLSubmit(response, request)
	}

	if admission.job == nil {
		t.Fatalf("admission not attempted: %d %s", response.Code, response.Body.String())
	}

	validateUncertainResponse(t, database, admission, response, delayed)

	reader, err := files.DownloadFile(ctx, artifacts.BucketStaging, admission.job.ID+"/"+expectedFile)
	if err != nil {
		t.Fatalf("uncertain commit deleted %s: %v", expectedFile, err)
	}

	data, err := io.ReadAll(reader)
	_ = reader.Close()

	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 || (input == "zip" && !bytes.Equal(data, zipBytes)) {
		t.Fatal("job input was corrupted")
	}

	assertUncertainJobRunnable(t, database, admission.job.ID)
}

func validateUncertainZIP(ctx context.Context, files *artifacts.Files, id string) error {
	file, err := files.OpenInput(ctx, id)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	return archive.Validate(file, info.Size())
}

func validateUncertainResponse(
	t *testing.T,
	database *jobstore.Database,
	admission *lostCommitAdmission,
	response *httptest.ResponseRecorder,
	delayed bool,
) {
	t.Helper()

	if delayed {
		if response.Code < 500 {
			t.Fatalf("uncertain absence was reported as success: %d", response.Code)
		}

		if admitErr := database.AdmitJob(context.Background(), admission.job); admitErr != nil {
			t.Fatal(admitErr)
		}

		return
	}

	if response.Code != http.StatusCreated {
		t.Fatalf("positive durable evidence not recovered: %d %s", response.Code, response.Body.String())
	}

	var body struct {
		JobID string `json:"job_id"`
	}
	if decodeErr := json.Unmarshal(response.Body.Bytes(), &body); decodeErr != nil || body.JobID != admission.job.ID {
		t.Fatal("recovered response did not identify persisted job")
	}
}
func assertUncertainJobRunnable(t *testing.T, database *jobstore.Database, id string) {
	t.Helper()

	ctx := context.Background()

	claimed, err := database.ClaimNext(ctx)
	if err != nil || claimed == nil || claimed.ID != id {
		t.Fatalf("persisted job could not run: %#v %v", claimed, err)
	}

	if markTerminalErr := database.MarkTerminal(
		ctx,
		claimed.ID,
		models.JobStateFailed,
		"",
		0,
		"test cleanup",
		"",
		"",
	); markTerminalErr != nil {
		t.Fatal(markTerminalErr)
	}

	if markCleanedErr := database.MarkCleaned(ctx, claimed.ID); markCleanedErr != nil {
		t.Fatal(markCleanedErr)
	}

	if deleteJobErr := database.DeleteJob(ctx, claimed.ID); deleteJobErr != nil {
		t.Fatal(deleteJobErr)
	}
}
