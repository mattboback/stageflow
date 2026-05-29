// Package metrics provides a small, dependency-free, thread-safe collector for
// orchestrator operational metrics. It renders Prometheus text-format output so
// the existing hand-written /metrics endpoint can expose process-level counters
// and a latency histogram without pulling in a metrics client dependency.
package metrics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// durationBucketsMs holds the upper bounds (inclusive, in milliseconds) for the
// event-handler duration histogram. The implicit +Inf bucket is emitted by
// WriteProm.
var durationBucketsMs = []float64{
	5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000, 60000,
}

type eventKey struct {
	event  string
	status string
}

// Collector accumulates orchestrator metrics in memory. The zero value is not
// usable; construct one with New. All methods are safe for concurrent use and
// tolerate a nil receiver so callers can stay unconditional.
type Collector struct {
	mu sync.Mutex

	eventCounts  map[eventKey]uint64
	httpStatus   map[int]uint64
	durationBkts []uint64 // cumulative-ready per-bucket counts (non-cumulative until rendered)
	durationSum  float64
	durationN    uint64
}

// New returns an initialized Collector.
func New() *Collector {
	return &Collector{
		eventCounts:  make(map[eventKey]uint64),
		httpStatus:   make(map[int]uint64),
		durationBkts: make([]uint64, len(durationBucketsMs)),
	}
}

// ObserveEvent records a single inbound-event handler outcome and its duration.
// status is the handler status recorded by the orchestrator ("ok", "error", or
// "panic").
func (c *Collector) ObserveEvent(event, status string, durationMs int64) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.eventCounts[eventKey{event: event, status: status}]++

	ms := float64(durationMs)
	for i, bound := range durationBucketsMs {
		if ms <= bound {
			c.durationBkts[i]++
		}
	}

	c.durationSum += ms
	c.durationN++
}

// ObserveHTTP records one admin-API response by status code.
func (c *Collector) ObserveHTTP(status int) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.httpStatus[status]++
}

// WriteProm appends the collected metrics to b in Prometheus text format.
func (c *Collector) WriteProm(b *strings.Builder) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.writeEventCounts(b)
	c.writeHTTPStatus(b)
	c.writeDurationHistogram(b)
}

func (c *Collector) writeEventCounts(b *strings.Builder) {
	b.WriteString("# HELP stageflow_orchestrator_event_handled_total Inbound events handled by outcome.\n")
	b.WriteString("# TYPE stageflow_orchestrator_event_handled_total counter\n")

	keys := make([]eventKey, 0, len(c.eventCounts))
	for k := range c.eventCounts {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].event != keys[j].event {
			return keys[i].event < keys[j].event
		}

		return keys[i].status < keys[j].status
	})

	for _, k := range keys {
		_, _ = fmt.Fprintf(
			b,
			"stageflow_orchestrator_event_handled_total{event=%q,status=%q} %d\n",
			k.event, k.status, c.eventCounts[k],
		)
	}
}

func (c *Collector) writeHTTPStatus(b *strings.Builder) {
	b.WriteString("# HELP stageflow_orchestrator_http_requests_total Admin API responses by status code.\n")
	b.WriteString("# TYPE stageflow_orchestrator_http_requests_total counter\n")

	codes := make([]int, 0, len(c.httpStatus))
	for code := range c.httpStatus {
		codes = append(codes, code)
	}

	sort.Ints(codes)

	for _, code := range codes {
		_, _ = fmt.Fprintf(
			b,
			"stageflow_orchestrator_http_requests_total{status=%q} %d\n",
			strconv.Itoa(code), c.httpStatus[code],
		)
	}
}

func (c *Collector) writeDurationHistogram(b *strings.Builder) {
	b.WriteString(
		"# HELP stageflow_orchestrator_event_handler_duration_milliseconds Event handler duration in milliseconds.\n",
	)
	b.WriteString("# TYPE stageflow_orchestrator_event_handler_duration_milliseconds histogram\n")

	for i, bound := range durationBucketsMs {
		_, _ = fmt.Fprintf(
			b,
			"stageflow_orchestrator_event_handler_duration_milliseconds_bucket{le=%q} %d\n",
			strconv.FormatFloat(bound, 'f', -1, 64), c.durationBkts[i],
		)
	}

	_, _ = fmt.Fprintf(
		b,
		"stageflow_orchestrator_event_handler_duration_milliseconds_bucket{le=\"+Inf\"} %d\n",
		c.durationN,
	)
	_, _ = fmt.Fprintf(
		b,
		"stageflow_orchestrator_event_handler_duration_milliseconds_sum %s\n",
		strconv.FormatFloat(c.durationSum, 'f', -1, 64),
	)
	_, _ = fmt.Fprintf(
		b,
		"stageflow_orchestrator_event_handler_duration_milliseconds_count %d\n",
		c.durationN,
	)
}
