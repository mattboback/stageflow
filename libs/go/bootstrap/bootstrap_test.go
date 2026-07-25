package bootstrap_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"

	"github.com/mattboback/stageflow/libs/go/bootstrap"
	"github.com/mattboback/stageflow/libs/go/config"
)

// The interesting behavior in this package is not "does it construct a client" —
// it is the IgnoreEnsureFailure contract, which deliberately returns a *usable
// client alongside a non-nil error*. That inverts Go's usual convention, so a
// caller that writes the idiomatic `if err != nil { return err }` silently drops a
// working client. Both branches are asserted for both constructors.

func TestSetupLogging(t *testing.T) {
	t.Parallel()

	logger := bootstrap.SetupLogging("test-service")
	if logger == nil {
		t.Fatal("SetupLogging returned nil")
	}

	// SetupLogging's side effect is the point: code that logs via the package-level
	// slog functions has to pick up the service-tagged logger.
	if slog.Default() == nil {
		t.Fatal("slog.Default() is nil after SetupLogging")
	}
}

// unreachableMinIO points at a port on loopback that nothing is listening on, so
// EnsureBuckets fails on dial rather than on credentials or on a bucket policy.
func unreachableMinIO(t *testing.T) config.MinIOConfig {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind TCP in this environment: %v", err)
	}

	addr := listener.Addr().String()

	if closeErr := listener.Close(); closeErr != nil {
		t.Fatalf("releasing probe listener: %v", closeErr)
	}

	return config.MinIOConfig{
		Endpoint:  addr,
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	}
}

func TestNewMinIOClient_InvalidEndpoint(t *testing.T) {
	t.Parallel()

	client, err := bootstrap.NewMinIOClient(
		context.Background(),
		config.MinIOConfig{Endpoint: "http://not a valid endpoint"},
		bootstrap.MinIOOptions{},
	)
	if err == nil {
		t.Fatal("expected an error for a malformed endpoint")
	}

	if client != nil {
		t.Error("expected no client when construction fails")
	}
}

func TestNewMinIOClient_SkipsEnsureWhenNotRequested(t *testing.T) {
	t.Parallel()

	// The endpoint is unreachable, so a client returned without error proves
	// EnsureBuckets was genuinely not attempted rather than attempted and ignored.
	client, err := bootstrap.NewMinIOClient(
		context.Background(),
		unreachableMinIO(t),
		bootstrap.MinIOOptions{EnsureBuckets: false},
	)
	if err != nil {
		t.Fatalf("expected no error when EnsureBuckets is off: %v", err)
	}

	if client == nil {
		t.Error("expected a client")
	}
}

func TestNewMinIOClient_EnsureFailure(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name          string
		ignore        bool
		wantClient    bool
		wantErrSubstr string
	}{
		{
			name:          "surfaces the failure and discards the client",
			ignore:        false,
			wantClient:    false,
			wantErrSubstr: "ensure MinIO buckets",
		},
		{
			// Deliberately both: the caller is handed a working client *and* told
			// bucket setup did not complete.
			name:       "returns the client alongside the failure when ignoring",
			ignore:     true,
			wantClient: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := unreachableMinIO(t)

			// A canceled context makes the dial fail immediately instead of waiting
			// out minio-go's retry budget, keeping the test fast and deterministic.
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			client, err := bootstrap.NewMinIOClient(ctx, cfg, bootstrap.MinIOOptions{
				EnsureBuckets:       true,
				IgnoreEnsureFailure: testCase.ignore,
			})
			if err == nil {
				t.Fatal("expected EnsureBuckets to fail against an unreachable endpoint")
			}

			if testCase.wantErrSubstr != "" && !strings.Contains(err.Error(), testCase.wantErrSubstr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErrSubstr)
			}

			if got := client != nil; got != testCase.wantClient {
				t.Errorf("client != nil = %v, want %v", got, testCase.wantClient)
			}
		})
	}
}

func TestNewNATSClient_UnreachableServer(t *testing.T) {
	t.Parallel()

	client, err := bootstrap.NewNATSClient(
		context.Background(),
		config.NATSConfig{
			URL:            "nats://127.0.0.1:1",
			ConnectTimeout: 100 * time.Millisecond,
			MaxReconnects:  0,
		},
		bootstrap.NATSOptions{},
	)
	if err == nil {
		if client != nil {
			_ = client.Close()
		}

		t.Fatal("expected an error connecting to a closed port")
	}

	if client != nil {
		t.Error("expected no client when the connection fails")
	}
}

func TestNewNATSClient_EnsureStreams(t *testing.T) {
	t.Parallel()

	// EnsureStreams needs JetStream. Running one server with it and one without
	// reaches the success and failure branches through the real client rather than
	// through a stub, which is what makes the IgnoreEnsureFailure assertions
	// meaningful.
	withJetStream := startEmbeddedNATS(t, true)
	withoutJetStream := startEmbeddedNATS(t, false)

	for _, testCase := range []struct {
		name          string
		url           string
		ignore        bool
		wantErr       bool
		wantClient    bool
		wantErrSubstr string
	}{
		{
			name:       "succeeds when JetStream is available",
			url:        withJetStream,
			wantErr:    false,
			wantClient: true,
		},
		{
			name:          "discards the client when stream setup fails",
			url:           withoutJetStream,
			ignore:        false,
			wantErr:       true,
			wantClient:    false,
			wantErrSubstr: "ensure NATS streams",
		},
		{
			name:       "returns the client alongside the failure when ignoring",
			url:        withoutJetStream,
			ignore:     true,
			wantErr:    true,
			wantClient: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client, err := bootstrap.NewNATSClient(
				context.Background(),
				config.NATSConfig{URL: testCase.url, ConnectTimeout: 5 * time.Second},
				bootstrap.NATSOptions{
					EnsureStreams:       true,
					IgnoreEnsureFailure: testCase.ignore,
				},
			)

			if client != nil {
				t.Cleanup(func() { _ = client.Close() })
			}

			if gotErr := err != nil; gotErr != testCase.wantErr {
				t.Fatalf("err != nil = %v (%v), want %v", gotErr, err, testCase.wantErr)
			}

			if testCase.wantErrSubstr != "" && !strings.Contains(err.Error(), testCase.wantErrSubstr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErrSubstr)
			}

			if got := client != nil; got != testCase.wantClient {
				t.Errorf("client != nil = %v, want %v", got, testCase.wantClient)
			}
		})
	}
}

// startEmbeddedNATS runs an in-process NATS server on an ephemeral port and
// returns its client URL. Mirrors the pattern in
// services/platform-api/tests/integration/messaging_nats_test.go; an ephemeral
// port is used here (rather than the default) so this module's tests cannot
// collide with libs/go/messaging's, which deliberately binds the default port.
func startEmbeddedNATS(t *testing.T, jetStream bool) string {
	t.Helper()

	opts := &natsserver.Options{
		Host:      "127.0.0.1",
		Port:      -1, // ephemeral
		JetStream: jetStream,
		NoLog:     true,
		NoSigs:    true,
	}

	if jetStream {
		storeDir, err := os.MkdirTemp("", "stageflow-bootstrap-jetstream-")
		if err != nil {
			t.Fatalf("creating JetStream store dir: %v", err)
		}

		t.Cleanup(func() { _ = os.RemoveAll(storeDir) })

		opts.StoreDir = storeDir
	}

	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Skipf("cannot start embedded NATS: %v", err)
	}

	go srv.Start()

	t.Cleanup(srv.Shutdown)

	if !srv.ReadyForConnections(10 * time.Second) {
		// A sandbox that forbids binding surfaces here as a startup timeout; skip
		// with the reason rather than failing on the environment.
		if bindErr := probeBind(); bindErr != nil {
			t.Skipf("cannot bind TCP in this environment: %v", bindErr)
		}

		t.Fatal("timed out waiting for embedded NATS to accept connections")
	}

	return srv.ClientURL()
}

func probeBind() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			return fmt.Errorf("binding is not permitted: %w", err)
		}

		return err
	}

	return listener.Close()
}
