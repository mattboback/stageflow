package podman

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestInspectNetworkDecodesOptions(t *testing.T) {
	server := newMockPodmanServer()
	server.handle(http.MethodGet, libpodPrefix+"/networks/stageflow_jobs/json",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"stageflow_jobs","driver":"bridge","options":{"isolate":"strict"}}`))
		})

	info, err := server.newClient().InspectNetwork(context.Background(), "stageflow_jobs")
	if err != nil {
		t.Fatalf("InspectNetwork() error = %v", err)
	}

	if info.Name != "stageflow_jobs" || info.Driver != "bridge" || info.Options["isolate"] != "strict" {
		t.Fatalf("InspectNetwork() = %+v, want the name, driver and isolate option", info)
	}
}

func TestInspectNetworkReportsMissingNetworkAsNotFound(t *testing.T) {
	_, err := newMockPodmanServer().newClient().InspectNetwork(context.Background(), "absent")
	if !IsNotFound(err) {
		t.Fatalf("InspectNetwork() error = %v, want a not-found API error", err)
	}
}

type stubNetworkInspector struct {
	info *NetworkInfo
	err  error
}

func (s stubNetworkInspector) InspectNetwork(context.Context, string) (*NetworkInfo, error) {
	return s.info, s.err
}

func TestVerifyJobNetwork(t *testing.T) {
	tests := []struct {
		name      string
		inspector stubNetworkInspector
		wantErr   string
	}{
		{
			name:      "accepts a strictly isolated network",
			inspector: stubNetworkInspector{info: &NetworkInfo{Options: map[string]string{"isolate": "strict"}}},
		},
		{
			name:      "refuses a plain bridge, which can reach the stack by address",
			inspector: stubNetworkInspector{info: &NetworkInfo{Driver: "bridge"}},
			wantErr:   `has isolate="", want "strict"`,
		},
		{
			name:      "refuses isolate=true, which only blocks networks that opt in",
			inspector: stubNetworkInspector{info: &NetworkInfo{Options: map[string]string{"isolate": "true"}}},
			wantErr:   `has isolate="true", want "strict"`,
		},
		{
			name:      "names the create command when the network is missing",
			inspector: stubNetworkInspector{err: &APIError{StatusCode: http.StatusNotFound}},
			wantErr:   "podman network create --opt isolate=strict stageflow_jobs",
		},
		{
			name:      "reports a Podman failure as such",
			inspector: stubNetworkInspector{err: &APIError{StatusCode: http.StatusInternalServerError, Body: "boom"}},
			wantErr:   `inspect job network "stageflow_jobs"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := VerifyJobNetwork(context.Background(), test.inspector, "stageflow_jobs")
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("VerifyJobNetwork() error = %v, want nil", err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("VerifyJobNetwork() error = %v, want it to contain %q", err, test.wantErr)
			}
		})
	}
}
