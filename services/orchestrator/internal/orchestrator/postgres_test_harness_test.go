package orchestrator

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

var testDatabaseURL string

func TestMain(m *testing.M) {
	baseDir, err := os.MkdirTemp("", "stageflow-epg-orchestrator-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create embedded postgres temp dir: %v\n", err)
		os.Exit(1)
	}

	port, err := allocateFreePort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "allocate postgres port: %v\n", err)

		_ = os.RemoveAll(baseDir)

		os.Exit(1)
	}

	cfg := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V18).
		CachePath(filepath.Join(baseDir, "cache")).
		RuntimePath(filepath.Join(baseDir, "runtime")).
		BinariesPath(filepath.Join(baseDir, "binaries")).
		DataPath(filepath.Join(baseDir, "data")).
		Port(uint32(port)).
		Database("stageflow_test").
		Username("stageflow").
		Password("stageflow")

	server := embeddedpostgres.NewDatabase(cfg)
	if startErr := server.Start(); startErr != nil {
		fmt.Fprintf(os.Stderr, "start embedded postgres: %v\n", startErr)

		_ = os.RemoveAll(baseDir)

		os.Exit(1)
	}

	testDatabaseURL = fmt.Sprintf(
		"postgres://stageflow:stageflow@127.0.0.1:%d/stageflow_test?sslmode=disable",
		port,
	)

	code := m.Run()

	if stopErr := server.Stop(); stopErr != nil {
		fmt.Fprintf(os.Stderr, "stop embedded postgres: %v\n", stopErr)
	}

	if cleanupErr := os.RemoveAll(baseDir); cleanupErr != nil {
		fmt.Fprintf(os.Stderr, "cleanup embedded postgres temp dir: %v\n", cleanupErr)
	}

	os.Exit(code)
}

func allocateFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = listener.Close()
	}()

	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener addr type: %T", listener.Addr())
	}

	return address.Port, nil
}
