package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

func (o *Orchestrator) maintainArtifacts(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		if err := o.retainArtifacts(ctx, time.Now()); err != nil {
			slog.Warn("Artifact retention failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (o *Orchestrator) retainArtifacts(ctx context.Context, now time.Time) error {
	jobs, err := o.database.TerminalJobsBefore(ctx, now.Add(-24*time.Hour))
	if err != nil {
		return err
	}

	root, err := os.OpenRoot(filepath.Join(o.dataDir, "jobs"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	defer root.Close()

	for _, job := range jobs {
		if retainErr := o.retainTerminalJob(ctx, root, job); retainErr != nil {
			return retainErr
		}
	}

	return o.retainOrphanDirectories(ctx, root, now)
}

func (o *Orchestrator) retainTerminalJob(ctx context.Context, root *os.Root, job *models.Job) error {
	// Pod/auth cleanup must finish before deleting its durable record.
	if err := o.CleanupJob(ctx, job); err != nil {
		return err
	}

	if err := o.database.DeleteJob(ctx, job.ID); err != nil {
		return err
	}

	return root.RemoveAll(job.ID)
}

func (o *Orchestrator) retainOrphanDirectories(ctx context.Context, root *os.Root, now time.Time) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()

	entries, err := dir.ReadDir(-1)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}

		if now.Sub(info.ModTime()) < time.Hour {
			continue
		}

		exists, existsErr := o.database.JobExists(ctx, entry.Name())
		if existsErr != nil {
			return existsErr
		}

		if !exists {
			if removeErr := root.RemoveAll(entry.Name()); removeErr != nil {
				return removeErr
			}
		}
	}

	return nil
}
