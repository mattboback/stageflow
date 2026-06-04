// Package orchtest provides shared test infrastructure for the orchestrator
// packages that exercise real PostgreSQL behavior. It is imported only from
// test code.
package orchtest

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// downloadLockTimeout bounds how long a process waits for another process to
// finish populating the shared binary cache before proceeding anyway.
const downloadLockTimeout = 5 * time.Minute

// PostgresForTests returns a PostgreSQL connection URL for tests plus a stop
// function that must be called when the suite finishes.
//
// Behavior is contributor-friendly by design:
//   - If TEST_DATABASE_URL is set, that database is used as-is. No binaries are
//     downloaded and no server is started, so contributors can run the suite
//     against an existing PostgreSQL (e.g. one started by `just dev up`).
//   - Otherwise an embedded PostgreSQL is started on a random free port. The
//     binary archive is cached in a stable shared directory so the (~100 MB)
//     download happens once per machine instead of once per package, per run.
//
// prefix names the per-run runtime directory so parallel packages do not
// collide on disk.
func PostgresForTests(prefix string) (url string, stop func() error, err error) {
	if external := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")); external != "" {
		return external, func() error { return nil }, nil
	}

	cacheRoot := sharedCacheDir()
	if mkErr := os.MkdirAll(cacheRoot, 0o750); mkErr != nil {
		return "", nil, fmt.Errorf("create embedded postgres cache dir: %w", mkErr)
	}

	baseDir, err := os.MkdirTemp("", "stageflow-epg-"+prefix+"-*")
	if err != nil {
		return "", nil, fmt.Errorf("create embedded postgres temp dir: %w", err)
	}

	port, err := allocateFreePort()
	if err != nil {
		_ = os.RemoveAll(baseDir)
		return "", nil, fmt.Errorf("allocate postgres port: %w", err)
	}

	cfg := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V18).
		CachePath(cacheRoot).
		RuntimePath(filepath.Join(baseDir, "runtime")).
		BinariesPath(filepath.Join(baseDir, "binaries")).
		DataPath(filepath.Join(baseDir, "data")).
		Port(port).
		Database("stageflow_test").
		Username("stageflow").
		Password("stageflow")

	// Serialize the first (cold-cache) download so parallel test packages do not
	// race to write the same archive. Once the cache is warm, packages start in
	// parallel without locking.
	var unlock func()
	if !cacheIsPopulated(cacheRoot) {
		unlock = acquireDownloadLock(cacheRoot)
	}

	server := embeddedpostgres.NewDatabase(cfg)
	if startErr := server.Start(); startErr != nil {
		if unlock != nil {
			unlock()
		}

		_ = os.RemoveAll(baseDir)

		return "", nil, fmt.Errorf("start embedded postgres: %w", startErr)
	}

	if unlock != nil {
		unlock()
	}

	url = fmt.Sprintf(
		"postgres://stageflow:stageflow@127.0.0.1:%d/stageflow_test?sslmode=disable",
		port,
	)

	stop = func() error {
		stopErr := server.Stop()
		_ = os.RemoveAll(baseDir)

		return stopErr
	}

	return url, stop, nil
}

// sharedCacheDir returns a stable directory for caching the embedded postgres
// binary archive across runs, preferring the user cache dir.
func sharedCacheDir() string {
	if userCache, err := os.UserCacheDir(); err == nil {
		return filepath.Join(userCache, "stageflow", "embedded-postgres")
	}

	return filepath.Join(os.TempDir(), "stageflow-embedded-postgres")
}

func cacheIsPopulated(cacheRoot string) bool {
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".txz") {
			return true
		}
	}

	return false
}

// acquireDownloadLock takes a cross-process lock via an atomically-created lock
// directory. It always returns a release function; on timeout it proceeds
// without the lock rather than failing the suite.
func acquireDownloadLock(cacheRoot string) func() {
	lockDir := filepath.Join(cacheRoot, ".download.lock")
	deadline := time.Now().Add(downloadLockTimeout)

	for {
		err := os.Mkdir(lockDir, 0o750)
		if err == nil {
			return func() { _ = os.Remove(lockDir) }
		}

		if !os.IsExist(err) || time.Now().After(deadline) {
			return func() {}
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func allocateFreePort() (uint32, error) {
	var lc net.ListenConfig

	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
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

	if address.Port < 0 || address.Port > 65535 {
		return 0, fmt.Errorf("allocated port out of range: %d", address.Port)
	}

	return uint32(address.Port), nil
}
