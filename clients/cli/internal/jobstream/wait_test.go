package jobstream

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

func TestWaitJobStateUsesSSEAndFallsBackWhenUnavailable(t *testing.T) {
	t.Parallel()

	const statusJSON = `{"id":"job-1","state":"DONE"}`

	for _, testCase := range []struct {
		name         string
		streamStatus int
		streamBody   string
		wantPaths    []string
		wantOutput   string
	}{
		{
			name:         "stream completion",
			streamStatus: http.StatusOK,
			streamBody:   "event: status\ndata: " + statusJSON + "\n\n",
			wantPaths:    []string{"/api/v1/jobs/job-1/stream"},
			wantOutput:   "state: DONE\n",
		},
		{
			name:         "stream unavailable",
			streamStatus: http.StatusNotFound,
			wantPaths:    []string{"/api/v1/jobs/job-1/stream", "/api/v1/jobs/job-1"},
			wantOutput:   "stream lost, falling back to polling...\nstate: DONE\n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var paths []string

			client := apiclient.NewClient("http://stageflow.test", "", &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					paths = append(paths, req.URL.Path)
					status := http.StatusOK
					body := statusJSON

					if strings.HasSuffix(req.URL.Path, "/stream") {
						status = testCase.streamStatus
						body = testCase.streamBody
					}

					return &http.Response{
						StatusCode: status,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(body)),
					}, nil
				}),
			})

			var output bytes.Buffer
			if err := WaitJobState(context.Background(), client, "job-1", &output); err != nil {
				t.Fatalf("WaitJobState: %v", err)
			}

			if !reflect.DeepEqual(paths, testCase.wantPaths) || output.String() != testCase.wantOutput {
				t.Fatalf(
					"paths=%v, output=%q; want %v, %q",
					paths, output.String(), testCase.wantPaths, testCase.wantOutput,
				)
			}
		})
	}
}
