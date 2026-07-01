package jobstatus

import (
	"context"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

type CurrentReader interface {
	GetJob(ctx context.Context, jobID string) (*status.JobRecord, error)
}

type Config struct {
	CurrentReader CurrentReader
	CacheTTL      time.Duration
}

type StatusPipeline interface {
	Begin(ctx context.Context, cmd BeginJob) (*status.JobRecord, error)
	Apply(ctx context.Context, signal Signal) (*status.JobRecord, error)
	Current(ctx context.Context, jobID string) (*status.JobRecord, error)
	Watch(ctx context.Context, jobID string) (*status.JobRecord, Subscription, error)
}

type BeginJob struct {
	Payload    *events.JobCreatedPayload
	ObservedAt time.Time
}

type SignalKind string

const (
	SignalJobCreated        SignalKind = events.EventJobCreated
	SignalExtractionReady   SignalKind = events.EventExtractionReady
	SignalExtractionFailed  SignalKind = events.EventExtractionFailed
	SignalScanPageCompleted SignalKind = events.EventScanPageCompleted
	SignalScanCompleted     SignalKind = events.EventScanCompleted
	SignalScanFailed        SignalKind = events.EventScanFailed
	SignalJobCompleted      SignalKind = events.EventJobCompleted
	SignalJobFailed         SignalKind = events.EventJobFailed
)

type Signal struct {
	Kind              SignalKind
	ObservedAt        time.Time
	JobCreated        *events.JobCreatedPayload
	ExtractionReady   *events.ExtractionReadyPayload
	ExtractionFailed  *events.ExtractionFailedPayload
	ScanPageCompleted *events.ScanPageCompletedPayload
	ScanCompleted     *events.ScanCompletedPayload
	ScanFailed        *events.ScanFailedPayload
	JobCompleted      *events.JobCompletedPayload
	JobFailed         *events.JobFailedPayload
}

type Change struct {
	JobID      string
	Snapshot   *status.JobRecord
	Signal     Signal
	ObservedAt time.Time
}

type Subscription interface {
	Updates() <-chan Change
	Close() error
}
