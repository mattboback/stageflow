package messaging

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
)

func (c *Client) createOrRefreshConsumer(
	ctx context.Context,
	stream, subject, consumerName string,
) (jetstream.Consumer, error) {
	js, err := c.snapshot()
	if err != nil {
		return nil, err
	}

	cfg := jetstream.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    defaultConsumerMaxDeliver,
		AckWait:       defaultConsumerAckWait,
	}

	cons, err := js.CreateOrUpdateConsumer(ctx, stream, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	streamHandle, err := js.Stream(ctx, stream)
	if err != nil {
		return nil, fmt.Errorf("failed to load stream %s: %w", stream, err)
	}

	streamInfo, err := streamHandle.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect stream %s: %w", stream, err)
	}

	consumerInfo, err := cons.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect consumer %s on stream %s: %w", consumerName, stream, err)
	}

	lastSubjectSeq := uint64(0)
	if rawMsg, msgErr := streamHandle.GetLastMsgForSubject(ctx, subject); msgErr == nil && rawMsg != nil {
		lastSubjectSeq = rawMsg.Sequence
	}

	if consumerStateLooksStale(streamInfo, consumerInfo, lastSubjectSeq) {
		slog.Warn(
			"Resetting stale JetStream consumer",
			"stream", stream,
			"subject", subject,
			"consumer", consumerName,
			"stream_last_seq", streamInfo.State.LastSeq,
			"subject_last_seq", lastSubjectSeq,
			"consumer_delivered_stream_seq", consumerInfo.Delivered.Stream,
			"consumer_ack_floor_stream_seq", consumerInfo.AckFloor.Stream,
			"consumer_num_pending", consumerInfo.NumPending,
		)

		if deleteErr := streamHandle.DeleteConsumer(ctx, consumerName); deleteErr != nil {
			return nil, fmt.Errorf(
				"failed to delete stale consumer %s on stream %s: %w",
				consumerName,
				stream,
				deleteErr,
			)
		}

		cons, err = js.CreateOrUpdateConsumer(ctx, stream, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate consumer %s on stream %s: %w", consumerName, stream, err)
		}
	}

	return cons, nil
}

func consumerStateLooksStale(
	streamInfo *jetstream.StreamInfo,
	consumerInfo *jetstream.ConsumerInfo,
	lastSubjectSeq uint64,
) bool {
	if streamInfo == nil || consumerInfo == nil {
		return false
	}

	lastSeq := streamInfo.State.LastSeq
	if lastSeq > 0 && (consumerInfo.Delivered.Stream > lastSeq || consumerInfo.AckFloor.Stream > lastSeq) {
		return true
	}

	if lastSubjectSeq == 0 {
		return false
	}

	consumerTail := consumerInfo.AckFloor.Stream
	if consumerInfo.Delivered.Stream > consumerTail {
		consumerTail = consumerInfo.Delivered.Stream
	}

	return lastSubjectSeq > consumerTail && consumerInfo.NumPending == 0
}
