package apiclient

import (
	"archive/zip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSubmitZipJob(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "site.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	zw := zip.NewWriter(f)

	entry, err := zw.Create("index.html")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := entry.Write([]byte("<html></html>")); err != nil {
		t.Fatal(err)
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	var gotFilename, gotModules, gotScreenshot string

	var gotFileBytes int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/jobs/zip" {
			t.Errorf("path = %s, want /api/v1/jobs/zip", r.URL.Path)
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("read file part: %v", err)
		}

		defer func() { _ = file.Close() }()

		buf := make([]byte, 1<<16)

		for {
			n, readErr := file.Read(buf)
			gotFileBytes += n

			if readErr != nil {
				break
			}
		}

		gotFilename = header.Filename
		gotModules = r.FormValue("modules")
		gotScreenshot = r.FormValue("screenshot")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"job_id": "job-zip-1", "status": "pending", "message": "ok",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "", nil)

	resp, err := client.SubmitZipJob(context.Background(), zipPath, []string{"axe", "seo"}, true)
	if err != nil {
		t.Fatalf("SubmitZipJob: %v", err)
	}

	if resp.JobID != "job-zip-1" {
		t.Fatalf("JobID = %q, want job-zip-1", resp.JobID)
	}

	if gotFilename != "site.zip" {
		t.Fatalf("filename = %q, want site.zip", gotFilename)
	}

	if gotFileBytes == 0 {
		t.Fatal("file part was empty")
	}

	if gotModules != "axe,seo" {
		t.Fatalf("modules = %q, want axe,seo", gotModules)
	}

	if gotScreenshot != "true" {
		t.Fatalf("screenshot = %q, want true", gotScreenshot)
	}
}

func TestSubmitZipJobServerError(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "site.zip")
	if err := os.WriteFile(zipPath, []byte("zipbytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"boom"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", nil)

	_, err := client.SubmitZipJob(context.Background(), zipPath, nil, false)
	if err == nil {
		t.Fatal("expected error on 400 response")
	}
}
