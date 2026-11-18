// sanitize-backup operates only on the private Unix socket made by backup-snapshot.sh.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattboback/stageflow/internal/jobstore"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sanitize-backup:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return errors.New("usage: sanitize-backup /private/stageflow-backup.XXXXXX/socket")
	}

	socket := filepath.Clean(os.Args[1])

	parent := filepath.Dir(socket)
	if !filepath.IsAbs(socket) || filepath.Base(socket) != "socket" ||
		!strings.HasPrefix(filepath.Base(parent), "stageflow-backup.") {
		return errors.New("refusing a socket outside an isolated backup directory")
	}

	for _, path := range []string{parent, socket} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}

		if !info.IsDir() || info.Mode().Perm() != 0o700 {
			return errors.New("backup directories must be private real directories")
		}
	}

	values := url.Values{
		"host":    {socket},
		"user":    {"stageflow_backup"},
		"dbname":  {"stageflow_backup"},
		"sslmode": {"disable"},
	}

	db, err := jobstore.NewDatabase(&jobstore.Config{URL: "postgres:///?" + values.Encode(), RequireSchema: true})
	if err != nil {
		return fmt.Errorf("open isolated snapshot: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return db.SanitizeBackup(ctx)
}
