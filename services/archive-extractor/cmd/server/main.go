// Package main runs an isolated static server over a validated uploaded ZIP.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/mattboback/stageflow/libs/go/config"
	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
	"github.com/mattboback/stageflow/services/archive-extractor/internal/discovery"
	"github.com/mattboback/stageflow/services/archive-extractor/internal/extractor"
	"github.com/mattboback/stageflow/services/archive-extractor/internal/provenance"
	"github.com/mattboback/stageflow/services/archive-extractor/internal/server"
)

type Config struct {
	JobID, InputPath, SiteDir, ProvenancePath, Port string
	NATS                                            config.NATSConfig
}

func loadConfig() *Config {
	return &Config{
		JobID:     config.GetEnv("JOB_ID", ""),
		InputPath: config.GetEnv("INPUT_PATH", "/job/input.zip"),
		SiteDir: config.GetEnv(
			"SITE_DIR",
			"/site/content",
		),
		ProvenancePath: config.GetEnv("PROVENANCE_PATH", "/provenance/provenance.json"),
		Port:           config.GetEnv("PORT", "8080"),
		NATS:           config.LoadNATSConfig(),
	}
}
func (c *Config) Validate() error {
	if c.JobID == "" {
		return errors.New("JOB_ID is required")
	}

	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("PORT must be between 1 and 65535")
	}

	return nil
}
func main() { os.Exit(run()) }

func run() int {
	cfg := loadConfig()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := cfg.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		return 1
	}

	if err := runExtraction(ctx, cfg); err != nil {
		slog.Error("Extraction failed", "error", err)
		return 1
	}

	return 0
}
func runExtraction(ctx context.Context, cfg *Config) (runErr error) {
	client, err := sharedmsg.NewClient(&cfg.NATS)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer func() {
		if runErr != nil {
			payload := &events.ExtractionFailedPayload{
				JobID:        cfg.JobID,
				Error:        "extraction failed",
				ErrorDetails: runErr.Error(),
			}
			if publishErr := client.PublishEvent(
				context.Background(),
				sharedmsg.SubjectExtractionFailed,
				events.NewEnvelope(events.EventExtractionFailed, cfg.JobID, "extractor", payload),
			); publishErr != nil {
				slog.Warn("Publish extraction failure", "error", publishErr)
			}
		}
	}()

	if extractErr := extractor.NewExtractor().Extract(ctx, cfg.InputPath, cfg.SiteDir); extractErr != nil {
		return extractErr
	}

	pages, err := discovery.DiscoverHTML(cfg.SiteDir)
	if err != nil {
		return err
	}

	siteServer := server.NewStaticServer(&server.Config{SiteDir: cfg.SiteDir, Port: cfg.Port})
	if startErr := siteServer.Start(ctx); startErr != nil {
		return startErr
	}

	defer func() {
		if stopErr := siteServer.Stop(context.Background()); stopErr != nil {
			slog.Warn("Stop static server", "error", stopErr)
		}
	}()

	baseURL := "http://" + siteServer.ListenerAddr()

	prov, err := provenance.NewGenerator().Generate(cfg.JobID, baseURL, pages, cfg.ProvenancePath)
	if err != nil {
		return err
	}

	payload := &events.ExtractionReadyPayload{
		JobID:                  cfg.JobID,
		ProvenancePath:         cfg.ProvenancePath,
		BaseURL:                baseURL,
		TotalPages:             len(prov.Pages),
		ProvenanceArtifactPath: cfg.JobID + "/provenance.json",
	}
	if publishErr := client.PublishEvent(
		ctx,
		sharedmsg.SubjectExtractionReady,
		events.NewEnvelope(events.EventExtractionReady, cfg.JobID, "extractor", payload),
	); publishErr != nil {
		return publishErr
	}

	return waitForServer(ctx, siteServer)
}

func waitForServer(ctx context.Context, siteServer *server.StaticServer) error {
	done := make(chan error, 1)
	go func() { done <- siteServer.Wait() }()

	select {
	case <-ctx.Done():
		return nil
	case err := <-done:
		if err != nil {
			return fmt.Errorf("static server lost after readiness: %w", err)
		}

		return errors.New("static server stopped before scanners finished")
	}
}
