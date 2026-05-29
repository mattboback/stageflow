package metrics

import (
	"strings"
	"testing"
)

func TestCollectorRendersEventCounters(t *testing.T) {
	c := New()
	c.ObserveEvent("job.created", "ok", 12)
	c.ObserveEvent("job.created", "ok", 30)
	c.ObserveEvent("job.created", "error", 5)
	c.ObserveEvent("scan.completed", "panic", 100)

	var b strings.Builder
	c.WriteProm(&b)
	out := b.String()

	wantLines := []string{
		`stageflow_orchestrator_event_handled_total{event="job.created",status="ok"} 2`,
		`stageflow_orchestrator_event_handled_total{event="job.created",status="error"} 1`,
		`stageflow_orchestrator_event_handled_total{event="scan.completed",status="panic"} 1`,
	}
	for _, line := range wantLines {
		if !strings.Contains(out, line) {
			t.Errorf("metrics output missing %q\n---\n%s", line, out)
		}
	}
}

func TestCollectorDurationHistogramIsCumulative(t *testing.T) {
	c := New()
	// Durations chosen to land in distinct buckets: 5ms (<=5), 40ms (<=50), 700ms (<=1000).
	c.ObserveEvent("job.created", "ok", 5)
	c.ObserveEvent("job.created", "ok", 40)
	c.ObserveEvent("job.created", "ok", 700)

	var b strings.Builder
	c.WriteProm(&b)
	out := b.String()

	// Cumulative buckets: le=5 -> 1, le=50 -> 2, le=1000 -> 3, +Inf -> 3.
	checks := map[string]string{
		`event_handler_duration_milliseconds_bucket{le="5"} 1`:    "le=5",
		`event_handler_duration_milliseconds_bucket{le="50"} 2`:   "le=50",
		`event_handler_duration_milliseconds_bucket{le="1000"} 3`: "le=1000",
		`event_handler_duration_milliseconds_bucket{le="+Inf"} 3`: "le=+Inf",
		`event_handler_duration_milliseconds_count 3`:             "count",
		`event_handler_duration_milliseconds_sum 745`:             "sum",
	}
	for line, label := range checks {
		if !strings.Contains(out, line) {
			t.Errorf("histogram missing %s line %q\n---\n%s", label, line, out)
		}
	}
}

func TestCollectorRendersHTTPStatus(t *testing.T) {
	c := New()
	c.ObserveHTTP(200)
	c.ObserveHTTP(200)
	c.ObserveHTTP(429)

	var b strings.Builder
	c.WriteProm(&b)
	out := b.String()

	for _, line := range []string{
		`stageflow_orchestrator_http_requests_total{status="200"} 2`,
		`stageflow_orchestrator_http_requests_total{status="429"} 1`,
	} {
		if !strings.Contains(out, line) {
			t.Errorf("http status output missing %q\n---\n%s", line, out)
		}
	}
}

func TestCollectorNilReceiverIsSafe(t *testing.T) {
	var c *Collector // nil

	// None of these should panic.
	c.ObserveEvent("job.created", "ok", 10)
	c.ObserveHTTP(200)

	var b strings.Builder
	c.WriteProm(&b)

	if b.Len() != 0 {
		t.Errorf("nil collector should render nothing, got %q", b.String())
	}
}
