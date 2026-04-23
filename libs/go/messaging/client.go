package messaging

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mattboback/stageflow/libs/go/config"
)

type Client struct {
	mu              sync.Mutex
	nc              *nats.Conn
	js              jetstream.JetStream
	closed          bool
	consumeContexts map[string]jetstream.ConsumeContext
}

type Config = config.NATSConfig

var (
	ErrNilClient       = errors.New("messaging: client is nil")
	ErrNilContext      = errors.New("messaging: context is nil")
	ErrNotConnected    = errors.New("messaging: client is not connected")
	ErrNilHandler      = errors.New("messaging: handler is nil")
	ErrBadSubscription = errors.New("messaging: invalid subscription")
	ErrBadEnvelope     = errors.New("messaging: invalid envelope")
)

func DefaultConfig() *Config {
	return &Config{
		URL:            nats.DefaultURL,
		MaxReconnects:  10,
		ReconnectWait:  2 * time.Second,
		ConnectTimeout: 10 * time.Second,
	}
}

func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Treat zero-values as "not set" (common when config is partially specified).
	normalized := *cfg
	if normalized.URL == "" {
		normalized.URL = nats.DefaultURL
	}

	if normalized.MaxReconnects == 0 {
		normalized.MaxReconnects = 10
	}

	if normalized.ReconnectWait == 0 {
		normalized.ReconnectWait = 2 * time.Second
	}

	if normalized.ConnectTimeout == 0 {
		normalized.ConnectTimeout = 10 * time.Second
	}

	opts := []nats.Option{
		nats.Name("stageflow"),
		nats.MaxReconnects(normalized.MaxReconnects),
		nats.ReconnectWait(normalized.ReconnectWait),
		nats.Timeout(normalized.ConnectTimeout),

		nats.ErrorHandler(func(_ *nats.Conn, sub *nats.Subscription, err error) {
			// sub can be nil in some error cases
			if sub != nil {
				slog.Error("NATS async error", "subject", sub.Subject, "queue", sub.Queue, "error", err)
			} else {
				slog.Error("NATS async error", "error", err)
			}
		}),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				slog.Warn("NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			slog.Info("NATS connection closed", "url", nc.ConnectedUrl())
		}),
	}

	nc, err := nats.Connect(normalized.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()

		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &Client{
		nc:              nc,
		js:              js,
		consumeContexts: make(map[string]jetstream.ConsumeContext),
	}, nil
}

// snapshot returns a safe copy of the JetStream handle under the mutex.
// Callers must use the returned handle instead of accessing c.js directly.
func (c *Client) snapshot() (jetstream.JetStream, error) {
	if c == nil {
		return nil, ErrNilClient
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrNotConnected
	}

	if c.nc == nil || c.nc.IsClosed() {
		return nil, ErrNotConnected
	}

	if c.js == nil {
		return nil, ErrNotConnected
	}

	return c.js, nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	c.mu.Lock()

	c.closed = true

	// Copy state under lock, then release before blocking I/O.
	contexts := make(map[string]jetstream.ConsumeContext, len(c.consumeContexts))
	for k, v := range c.consumeContexts {
		contexts[k] = v
	}

	c.consumeContexts = nil

	nc := c.nc
	c.nc = nil
	c.js = nil

	c.mu.Unlock()

	for _, consumeCtx := range contexts {
		consumeCtx.Stop()
	}

	if nc != nil {
		nc.Close()
	}

	return nil
}

func (c *Client) trackConsumeContext(stream, consumerName string, consumeCtx jetstream.ConsumeContext) {
	if c == nil || consumeCtx == nil {
		return
	}

	key := stream + "/" + consumerName

	c.mu.Lock()
	defer c.mu.Unlock()

	if prev, ok := c.consumeContexts[key]; ok {
		prev.Stop()
	}

	c.consumeContexts[key] = consumeCtx
}

func (c *Client) untrackConsumeContext(stream, consumerName string, consumeCtx jetstream.ConsumeContext) {
	if c == nil || consumeCtx == nil {
		return
	}

	key := stream + "/" + consumerName

	c.mu.Lock()
	defer c.mu.Unlock()

	if current, ok := c.consumeContexts[key]; ok && current == consumeCtx {
		delete(c.consumeContexts, key)
	}
}
