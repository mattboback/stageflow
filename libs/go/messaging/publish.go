package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func (c *Client) Publish(ctx context.Context, subject string, data any) error {
	if err := c.ensureReady(); err != nil {
		return err
	}

	if ctx == nil {
		return ErrNilContext
	}

	if subject == "" {
		return errors.New("messaging: subject is required")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = c.js.Publish(ctx, subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish to %s: %w", subject, err)
	}

	return nil
}

func (c *Client) PublishEvent(ctx context.Context, subject string, envelope any) error {
	if err := c.ensureReady(); err != nil {
		return err
	}

	if ctx == nil {
		return ErrNilContext
	}

	if err := validatePublishEventEnvelope(envelope); err != nil {
		return err
	}

	return c.Publish(ctx, subject, envelope)
}
