package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamJobs       = "jobs"
	StreamExtraction = "extraction"
	StreamScan       = "scan"
)

const (
	defaultStreamMaxAge       = 72 * time.Hour
	defaultConsumerAckWait    = 10 * time.Minute
	defaultConsumerMaxDeliver = 10
	defaultConsumerNAKDelay   = 5 * time.Second
)

const (
	SubjectJobCreated        = "jobs.events.created"
	SubjectJobCompleted      = "jobs.events.completed"
	SubjectJobFailed         = "jobs.events.failed"
	SubjectExtractionReady   = "extraction.events.ready"
	SubjectExtractionFailed  = "extraction.events.failed"
	SubjectScanPageCompleted = "scan.events.page.completed"
	SubjectScanCompleted     = "scan.events.completed"
	SubjectScanFailed        = "scan.events.failed"
)

func (c *Client) EnsureStreams(ctx context.Context) error {
	if c == nil {
		return ErrNilClient
	}

	if ctx == nil {
		return ErrNilContext
	}

	js, err := c.snapshot()
	if err != nil {
		return err
	}

	streams := []struct {
		name     string
		subjects []string
	}{
		{
			name: StreamJobs,
			subjects: []string{
				SubjectJobCreated,
				SubjectJobCompleted,
				SubjectJobFailed,
			},
		},
		{
			name: StreamExtraction,
			subjects: []string{
				SubjectExtractionReady,
				SubjectExtractionFailed,
			},
		},
		{
			name: StreamScan,
			subjects: []string{
				SubjectScanPageCompleted,
				SubjectScanCompleted,
				SubjectScanFailed,
			},
		},
	}

	for _, s := range streams {
		_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
			Name:      s.name,
			Subjects:  s.subjects,
			Retention: jetstream.LimitsPolicy,
			MaxAge:    defaultStreamMaxAge,
			Storage:   jetstream.FileStorage,
		})
		if err != nil {
			return fmt.Errorf("failed to create stream %s: %w", s.name, err)
		}
	}

	return nil
}
