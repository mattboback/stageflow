package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mattboback/stageflow/libs/go/models"
)

// handleMetrics exposes a small Prometheus-compatible operational snapshot.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	var b strings.Builder
	b.WriteString("# HELP stageflow_orchestrator_jobs_total Total jobs known by the orchestrator.\n")
	b.WriteString("# TYPE stageflow_orchestrator_jobs_total gauge\n")

	totalJobs, err := s.database.CountJobs(r.Context(), nil)
	if err != nil {
		slog.Warn("Failed to count total jobs for metrics", "error", err)

		totalJobs = 0
	}

	_, _ = fmt.Fprintf(&b, "stageflow_orchestrator_jobs_total %d\n", totalJobs)

	b.WriteString("# HELP stageflow_orchestrator_jobs_by_state Jobs by current state.\n")
	b.WriteString("# TYPE stageflow_orchestrator_jobs_by_state gauge\n")

	for _, state := range models.AllJobStates {
		count, countErr := s.database.CountJobs(r.Context(), &state)
		if countErr != nil {
			slog.Warn("Failed to count jobs by state for metrics", "state", state, "error", countErr)

			count = 0
		}

		_, _ = fmt.Fprintf(&b, "stageflow_orchestrator_jobs_by_state{state=%q} %d\n", state, count)
	}

	if s.podmanClient != nil {
		pods, listErr := s.podmanClient.ListPods(r.Context())
		if listErr != nil {
			slog.Warn("Failed to list pods for metrics", "error", listErr)
		} else {
			b.WriteString("# HELP stageflow_orchestrator_pods_total Pods visible to the orchestrator.\n")
			b.WriteString("# TYPE stageflow_orchestrator_pods_total gauge\n")
			_, _ = fmt.Fprintf(&b, "stageflow_orchestrator_pods_total %d\n", len(pods))

			statusCounts := map[string]int{}
			for _, pod := range pods {
				statusCounts[pod.Status]++
			}

			b.WriteString("# HELP stageflow_orchestrator_pods_by_status Pods by runtime status.\n")
			b.WriteString("# TYPE stageflow_orchestrator_pods_by_status gauge\n")

			for status, count := range statusCounts {
				_, _ = fmt.Fprintf(&b, "stageflow_orchestrator_pods_by_status{status=%q} %d\n", status, count)
			}
		}
	}

	s.metrics.WriteProm(&b)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, writeErr := w.Write([]byte(b.String())); writeErr != nil {
		slog.Warn("Failed to write metrics response", "error", writeErr)
	}
}
