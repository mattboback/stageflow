package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestDiscoverCommandPrintsSharedAPIJSON(t *testing.T) {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/discover" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}

		if body["url"] != "https://example.com" {
			t.Errorf("seed=%s", body["url"])
		}

		calls++

		writeJSONResponse(
			t,
			w,
			http.StatusOK,
			apiclient.DiscoveryResult{
				URLs:     []string{"https://example.com/", "https://example.com/about"},
				Source:   "sitemap",
				Sources:  []string{"sitemap", "crawl"},
				Complete: true,
			},
		)
	}))
	defer server.Close()

	stdout, stderr, code := runCLI(t, "stageflow", "--api", server.URL, "discover", "example.com", "--format", "json")
	if code != 0 {
		t.Fatalf("code%d stderr%s", code, stderr)
	}

	var result apiclient.DiscoveryResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}

	if calls != 1 || len(result.URLs) != 2 || !result.Complete {
		t.Fatalf("calls=%d result=%+v", calls, result)
	}
}

func TestScanDiscoverSubmitsOnlyResolvedPages(t *testing.T) {
	var submitted apiclient.SubmitJobRequest

	calls := []string{}
	jobID := "discovered-job"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/discover":
			writeJSONResponse(
				t,
				w,
				http.StatusOK,
				apiclient.DiscoveryResult{
					URLs:     []string{"https://example.com/", "https://example.com/about"},
					Source:   "crawl",
					Complete: false,
					Warnings: []string{"Depth limit reached"},
				},
			)
		case "/api/v1/jobs/urls":
			if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
				t.Error(err)
			}

			writeJSONResponse(t, w, http.StatusCreated, apiclient.SubmitJobResponse{JobID: jobID})
		case "/api/v1/jobs/" + jobID:
			writeJSONResponse(t, w, http.StatusOK, apiclient.JobStatus{ID: jobID, State: apiclient.JobStateDone})
		case "/api/v1/jobs/" + jobID + "/results":
			writeJSONResponse(t, w, http.StatusOK, testsupport.SampleReport(jobID))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, stderr, code := runCLI(t, "stageflow", "--api", server.URL, "scan", "https://example.com", "--discover")
	if code != 0 {
		t.Fatalf("code%d stderr%s", code, stderr)
	}

	if len(calls) < 2 || calls[0] != "/api/v1/discover" || calls[1] != "/api/v1/jobs/urls" ||
		!slices.Equal(submitted.URLs, []string{"https://example.com/", "https://example.com/about"}) ||
		submitted.Auth != nil {
		t.Fatalf("calls=%v submission=%+v", calls, submitted)
	}

	if !strings.Contains(stderr, "coverage is incomplete") {
		t.Fatalf("missing warning: %s", stderr)
	}
}

func TestScanDiscoverRejectsAuthPrivateAndMultipleSeedsBeforeRequests(t *testing.T) {
	calls := 0

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(http.StatusInternalServerError) },
		),
	)
	defer server.Close()

	for _, args := range [][]string{
		{"scan", "https://example.com", "--discover", "--auth-state", "missing.json"},
		{"scan", "https://example.com", "--discover", "--auth-recipe", "missing.json"},
		{"scan", "https://example.com", "--discover", "--allow-private-targets"},
		{"scan", "https://example.com", "https://example.com/about", "--discover"},
		{"scan", "http://127.0.0.1", "--discover"},
		{"discover", "https://user:secret@example.com"},
	} {
		_, _, code := runCLI(t, append([]string{"stageflow", "--api", server.URL}, args...)...)
		if code != 2 {
			t.Errorf("args=%v code=%d", args, code)
		}
	}

	if calls != 0 {
		t.Fatalf("unexpected requests%d", calls)
	}
}

func TestScanDiscoverEmptyResultDoesNotCreateJob(t *testing.T) {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++

		writeJSONResponse(
			t,
			w,
			http.StatusOK,
			apiclient.DiscoveryResult{URLs: []string{}, Complete: false, Warnings: []string{"Blocked by robots"}},
		)
	}))
	defer server.Close()

	_, _, code := runCLI(t, "stageflow", "--api", server.URL, "scan", "https://example.com", "--discover")
	if code != 2 || calls != 1 {
		t.Fatalf("code=%d calls=%d", code, calls)
	}
}
