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
	mu                sync.Mutex
	consumerStopMu    sync.Mutex
	consumeSetupWG    sync.WaitGroup
	nc                *nats.Conn
	js                jetstream.JetStream
	closed            bool
	consumersStopping bool
	consumeContexts   map[string]consumeContext
}

// consumeContext is the lifecycle surface used from JetStream subscriptions.
// Keeping the small interface here makes the shutdown barrier independently
// testable without a live NATS server.
type consumeContext interface {
	Stop()
	Closed() <-chan struct{}
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
		consumeContexts: make(map[string]consumeContext),
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

	c.StopConsumers()

	c.mu.Lock()

	c.closed = true

	nc := c.nc
	c.nc = nil
	c.js = nil

	c.mu.Unlock()

	if nc != nil {
		nc.Close()
	}

	return nil
}

// beginConsumeSetup registers subscription setup before shutdown closes the
// registration gate. StopConsumers waits for every registered setup to either
// fail or publish its ConsumeContext into consumeContexts.
func (c *Client) beginConsumeSetup() error {
	if c == nil {
		return ErrNilClient
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.consumersStopping || c.nc == nil || c.nc.IsClosed() || c.js == nil {
		return ErrNotConnected
	}

	c.consumeSetupWG.Add(1)

	return nil
}

func (c *Client) finishConsumeSetup() {
	if c != nil {
		c.consumeSetupWG.Done()
	}
}

// StopConsumers stops accepting subscriptions and waits until JetStream has
// finished all callbacks that were already in progress. Callers can therefore
// safely tear down callback dependencies after this method returns.
func (c *Client) StopConsumers() {
	if c == nil {
		return
	}

	// Serialize callers so a concurrent Close cannot race the same contexts.
	c.consumerStopMu.Lock()
	defer c.consumerStopMu.Unlock()

	c.mu.Lock()
	c.consumersStopping = true
	c.mu.Unlock()

	// The stopping flag prevents Add calls after this Wait begins.
	c.consumeSetupWG.Wait()

	c.mu.Lock()

	contexts := make(map[string]consumeContext, len(c.consumeContexts))
	for key, consumeCtx := range c.consumeContexts {
		contexts[key] = consumeCtx
	}
	c.mu.Unlock()

	for _, consumeCtx := range contexts {
		consumeCtx.Stop()
	}

	for _, consumeCtx := range contexts {
		<-consumeCtx.Closed()
	}

	c.mu.Lock()
	for key, consumeCtx := range contexts {
		if current, ok := c.consumeContexts[key]; ok && current == consumeCtx {
			delete(c.consumeContexts, key)
		}
	}
	c.mu.Unlock()
}

func (c *Client) trackConsumeContext(stream, consumerName string, consumeCtx consumeContext) {
	if c == nil || consumeCtx == nil {
		return
	}

	key := stream + "/" + consumerName

	c.mu.Lock()

	if c.consumeContexts == nil {
		c.consumeContexts = make(map[string]consumeContext)
	}

	previous := c.consumeContexts[key]

	c.consumeContexts[key] = consumeCtx
	c.mu.Unlock()

	if previous != nil && previous != consumeCtx {
		previous.Stop()
		<-previous.Closed()
	}
}

func (c *Client) untrackConsumeContext(stream, consumerName string, consumeCtx consumeContext) {
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
