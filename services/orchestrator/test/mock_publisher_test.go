package test

import (
	"context"

	"github.com/mattboback/stageflow/libs/go/events"
)

// mockPublisher implements Publisher interface for testing.
type mockPublisher struct {
	completedEvents []*events.JobCompletedPayload
	failedEvents    []*events.JobFailedPayload
}

func newMockPublisher() *mockPublisher {
	return &mockPublisher{
		completedEvents: make([]*events.JobCompletedPayload, 0),
		failedEvents:    make([]*events.JobFailedPayload, 0),
	}
}

func (m *mockPublisher) PublishJobCompleted(_ context.Context, payload *events.JobCompletedPayload) error {
	m.completedEvents = append(m.completedEvents, payload)
	return nil
}

func (m *mockPublisher) PublishJobFailed(_ context.Context, payload *events.JobFailedPayload) error {
	m.failedEvents = append(m.failedEvents, payload)
	return nil
}
