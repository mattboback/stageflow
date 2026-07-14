package apiclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchReportEscapesJobIDAndSanitizesLegacyGrade(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI != "/api/v1/jobs/job%2Fpart/results" {
			t.Errorf("RequestURI = %q", r.RequestURI)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"version":"2.0.0","meta":{"jobId":"job/part"},"summary":{"byScanner":{},"bySeverity":{"critical":0,"serious":0,"moderate":0,"minor":0},"pagesScanned":0,"pagesWithIssues":0,"score":100,"scoreGrade":"Excellent","totalIssues":0},"scanners":[],"pages":[],"issues":[]}`)
	}))
	defer server.Close()

	doc, err := NewClient(server.URL, "", server.Client()).FetchReport(context.Background(), "job/part")
	if err != nil {
		t.Fatalf("FetchReport: %v", err)
	}
	if doc.Summary.ScoreGrade == nil || *doc.Summary.ScoreGrade != "A+" {
		t.Fatalf("score grade = %v", doc.Summary.ScoreGrade)
	}
}

func TestSanitizeReportJSONLeavesCurrentGradesUntouched(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"scoreGrade":"A"}`)
	if got := string(SanitizeReportJSON(raw)); !strings.Contains(got, `"scoreGrade":"A"`) {
		t.Fatalf("SanitizeReportJSON = %q", got)
	}
}
