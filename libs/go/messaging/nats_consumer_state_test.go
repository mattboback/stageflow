package messaging

import (
	"testing"

	"github.com/nats-io/nats.go/jetstream"
)

func TestConsumerStateLooksStale_FalseWhenWithinStreamBounds(t *testing.T) {
	t.Parallel()

	streamInfo := &jetstream.StreamInfo{
		State: jetstream.StreamState{LastSeq: 12},
	}
	consumerInfo := &jetstream.ConsumerInfo{
		Delivered: jetstream.SequenceInfo{Stream: 12},
		AckFloor:  jetstream.SequenceInfo{Stream: 12},
	}

	if consumerStateLooksStale(streamInfo, consumerInfo, 12) {
		t.Fatal("expected consumer state to be healthy")
	}
}

func TestConsumerStateLooksStale_TrueWhenDeliveredPastStreamTail(t *testing.T) {
	t.Parallel()

	streamInfo := &jetstream.StreamInfo{
		State: jetstream.StreamState{LastSeq: 12},
	}
	consumerInfo := &jetstream.ConsumerInfo{
		Delivered: jetstream.SequenceInfo{Stream: 14},
		AckFloor:  jetstream.SequenceInfo{Stream: 14},
	}

	if !consumerStateLooksStale(streamInfo, consumerInfo, 12) {
		t.Fatal("expected stale consumer state when delivered sequence exceeds stream tail")
	}
}

func TestConsumerStateLooksStale_TrueWhenAckFloorPastStreamTail(t *testing.T) {
	t.Parallel()

	streamInfo := &jetstream.StreamInfo{
		State: jetstream.StreamState{LastSeq: 3},
	}
	consumerInfo := &jetstream.ConsumerInfo{
		Delivered: jetstream.SequenceInfo{Stream: 3},
		AckFloor:  jetstream.SequenceInfo{Stream: 4},
	}

	if !consumerStateLooksStale(streamInfo, consumerInfo, 3) {
		t.Fatal("expected stale consumer state when ack floor exceeds stream tail")
	}
}

func TestConsumerStateLooksStale_TrueWhenSubjectAdvancedWithoutPendingDelivery(t *testing.T) {
	t.Parallel()

	streamInfo := &jetstream.StreamInfo{
		State: jetstream.StreamState{LastSeq: 13},
	}
	consumerInfo := &jetstream.ConsumerInfo{
		Delivered:  jetstream.SequenceInfo{Stream: 12},
		AckFloor:   jetstream.SequenceInfo{Stream: 12},
		NumPending: 0,
	}

	if !consumerStateLooksStale(streamInfo, consumerInfo, 13) {
		t.Fatal("expected stale consumer state when subject advanced but consumer has no pending delivery")
	}
}
