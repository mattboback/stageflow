package messaging

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// A single embedded NATS server backs the client tests that need a real
// connection, mirroring the pattern already used by
// services/platform-api/tests/integration/messaging_nats_test.go.
//
// It listens on the NATS default port rather than an ephemeral one because the
// behavior under test is precisely that NewClient falls back to nats.DefaultURL
// when given a nil config or a Config with a zero-value URL. Client does not
// retain its normalized config, so the connection is the only observable proof
// that the fallback happened.
var (
	embeddedNATS    *natsserver.Server
	embeddedNATSErr error
)

func TestMain(m *testing.M) {
	embeddedNATS, embeddedNATSErr = startEmbeddedNATSOnDefaultPort()

	code := m.Run()

	if embeddedNATS != nil {
		embeddedNATS.Shutdown()
	}

	os.Exit(code)
}

// requireDefaultURLNATS skips when no server could be started on the default
// port, which happens in sandboxes that disallow binding and on machines already
// running NATS. The skip carries the reason, so it is never silent.
func requireDefaultURLNATS(t *testing.T) {
	t.Helper()

	if embeddedNATSErr != nil {
		t.Skipf("no embedded NATS on %s: %v", nats.DefaultURL, embeddedNATSErr)
	}
}

func startEmbeddedNATSOnDefaultPort() (*natsserver.Server, error) {
	// Probe first so a restricted sandbox produces a clear reason rather than an
	// opaque server start failure.
	listenCfg := &net.ListenConfig{}
	addr := fmt.Sprintf("127.0.0.1:%d", nats.DefaultPort)

	probe, err := listenCfg.Listen(context.Background(), "tcp", addr)
	if err != nil {
		switch {
		case errors.Is(err, syscall.EPERM), errors.Is(err, syscall.EACCES):
			return nil, fmt.Errorf("cannot bind TCP in this environment: %w", err)
		case errors.Is(err, syscall.EADDRINUSE):
			return nil, fmt.Errorf("port %d already in use: %w", nats.DefaultPort, err)
		default:
			return nil, fmt.Errorf("probing %s: %w", addr, err)
		}
	}

	if closeErr := probe.Close(); closeErr != nil {
		return nil, fmt.Errorf("releasing probe listener: %w", closeErr)
	}

	storeDir, err := os.MkdirTemp("", "stageflow-messaging-jetstream-")
	if err != nil {
		return nil, fmt.Errorf("creating JetStream store dir: %w", err)
	}

	srv, err := natsserver.NewServer(&natsserver.Options{
		Host:      "127.0.0.1",
		Port:      nats.DefaultPort,
		JetStream: true,
		StoreDir:  storeDir,
		NoLog:     true,
		NoSigs:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("creating embedded NATS server: %w", err)
	}

	go srv.Start()

	if !srv.ReadyForConnections(10 * time.Second) {
		srv.Shutdown()

		return nil, errors.New("timed out waiting for embedded NATS to accept connections")
	}

	return srv, nil
}
