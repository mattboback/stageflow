// Package main starts the extractor job runner.
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mattboback/stageflow/libs/go/config"
	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/archive-extractor/internal/discovery"
	"github.com/mattboback/stageflow/services/archive-extractor/internal/extractor"
	"github.com/mattboback/stageflow/services/archive-extractor/internal/provenance"
	"github.com/mattboback/stageflow/services/archive-extractor/internal/server"
)

func main() {
	slog.Info("Starting Static Server")

	cfg := loadConfig()

	if err := cfg.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runExtraction(ctx, cfg)
}

func runExtraction(ctx context.Context, cfg *Config) {
	natsClient := mustNATSClient(ctx, cfg)
	defer closeNATSClient(natsClient)

	slog.Info("Starting extraction and site preparation", "job_id", cfg.JobID)

	minioClient := mustMinIOClient(ctx, cfg, natsClient)

	stageLog := mustStageLogger(ctx, cfg, natsClient, minioClient)

	siteDir := mustExtractZIP(ctx, cfg, natsClient, stageLog, minioClient)

	pages := mustDiscoverPages(ctx, cfg, natsClient, stageLog, siteDir)

	prov, baseURL, provenancePath := mustGenerateProvenance(ctx, cfg, natsClient, stageLog, pages)

	provenanceArtifactPath := mustUploadProvenance(ctx, cfg, natsClient, stageLog, minioClient, provenancePath)

	siteServer := mustStartStaticServer(ctx, cfg, natsClient, stageLog, siteDir)
	defer stopStaticServer(ctx, siteServer)

	publishExtractionReady(ctx, cfg, natsClient, stageLog, prov, baseURL, provenancePath, provenanceArtifactPath)

	slog.Info("Static server ready and waiting for scanner", "job_id", cfg.JobID)
	waitForShutdown(ctx, cfg, siteServer, natsClient, stageLog)

	slog.Info("Static server stopped", "job_id", cfg.JobID)
}

func mustNATSClient(ctx context.Context, cfg *Config) *sharedmsg.Client {
	natsClient, err := sharedmsg.NewClient(&cfg.NATS)
	if err != nil {
		publishFailureAndExit(ctx, cfg, nil, nil, "nats", fmt.Sprintf("Failed to connect to NATS: %v", err))
	}

	return natsClient
}

func closeNATSClient(natsClient *sharedmsg.Client) {
	if err := natsClient.Close(); err != nil {
		slog.Error("Failed to close NATS client", "error", err)
	}
}

func mustMinIOClient(ctx context.Context, cfg *Config, natsClient *sharedmsg.Client) *storage.MinIOClient {
	minioClient, err := storage.NewMinIOClient(&cfg.MinIO)
	if err != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			nil,
			"extraction",
			fmt.Sprintf("Failed to create MinIO client: %v", err),
		)
	}

	return minioClient
}

func mustStageLogger(
	ctx context.Context,
	cfg *Config,
	natsClient *sharedmsg.Client,
	minioClient *storage.MinIOClient,
) *stageLogger {
	stageLog, err := newStageLogger(ctx, cfg, minioClient)
	if err != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			nil,
			"logging",
			fmt.Sprintf("Failed to initialize stage logger: %v", err),
		)
	}

	return stageLog
}

func mustExtractZIP(
	ctx context.Context,
	cfg *Config,
	natsClient *sharedmsg.Client,
	stageLog *stageLogger,
	minioClient *storage.MinIOClient,
) string {
	slog.Info("Extracting ZIP from MinIO", "job_id", cfg.JobID, "input_path", cfg.InputPath)

	ext := extractor.NewExtractor(minioClient)

	siteDir := filepath.Join(cfg.Workspace, "site")
	if err := ext.Extract(ctx, storage.BucketStaging, cfg.InputPath, siteDir); err != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			stageLog,
			"extraction",
			fmt.Sprintf("Failed to extract ZIP: %v", err),
		)
	}

	stageLog.recordEvent("zip_extracted", map[string]any{"workspace": siteDir})

	if err := ensureSitePermissions(siteDir); err != nil {
		slog.Warn("Failed to ensure site permissions", "error", err)
	}

	slog.Info("ZIP extracted", "job_id", cfg.JobID, "site_dir", siteDir)

	return siteDir
}

func ensureSitePermissions(siteDir string) error {
	// Best-effort: ensure the embedded file server can always read the extracted site.
	return filepath.WalkDir(siteDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			//nolint:gosec // Extracted sites should not be world-readable, only accessible via HTTP.
			return os.Chmod(path, 0o750)
		}

		return os.Chmod(path, 0o600)
	})
}

func mustDiscoverPages(
	ctx context.Context,
	cfg *Config,
	natsClient *sharedmsg.Client,
	stageLog *stageLogger,
	siteDir string,
) []discovery.HTMLPage {
	slog.Info("Discovering HTML pages", "job_id", cfg.JobID, "site_dir", siteDir)

	pages, err := discovery.DiscoverHTML(siteDir)
	if err != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			stageLog,
			"discovery",
			fmt.Sprintf("Failed to discover HTML: %v", err),
		)
	}

	slog.Info("Discovered HTML pages", "job_id", cfg.JobID, "page_count", len(pages))
	stageLog.setPagesDiscovered(len(pages))
	stageLog.recordEvent("discovery_completed", map[string]any{"pages": len(pages)})

	return pages
}

func mustGenerateProvenance(
	ctx context.Context,
	cfg *Config,
	natsClient *sharedmsg.Client,
	stageLog *stageLogger,
	pages []discovery.HTMLPage,
) (prov *models.Provenance, baseURL, provenancePath string) {
	slog.Info("Generating provenance.json", "job_id", cfg.JobID)

	provenanceGen := provenance.NewGenerator()
	baseURL = "http://localhost:" + cfg.Port
	provenancePath = cfg.Workspace + "/provenance.json"

	prov, err := provenanceGen.Generate(cfg.JobID, baseURL, pages, provenancePath)
	if err != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			stageLog,
			"provenance",
			fmt.Sprintf("Failed to generate provenance: %v", err),
		)
	}

	slog.Info("Provenance generated", "job_id", cfg.JobID, "page_count", len(prov.Pages))
	stageLog.setArtifacts(provenancePath, baseURL)
	stageLog.recordEvent("provenance_generated", map[string]any{"pages": len(prov.Pages)})

	return prov, baseURL, provenancePath
}

func mustUploadProvenance(
	ctx context.Context,
	cfg *Config,
	natsClient *sharedmsg.Client,
	stageLog *stageLogger,
	minioClient *storage.MinIOClient,
	provenancePath string,
) string {
	provenanceArtifactPath := cfg.JobID + "/provenance.json"
	slog.Info("Uploading provenance to MinIO", "job_id", cfg.JobID, "artifact_path", provenanceArtifactPath)

	provFile, err := os.Open(provenancePath)
	if err != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			stageLog,
			"provenance_upload",
			fmt.Sprintf("Failed to open provenance file: %v", err),
		)
	}

	defer func() { _ = provFile.Close() }()

	provInfo, err := provFile.Stat()
	if err != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			stageLog,
			"provenance_upload",
			fmt.Sprintf("Failed to stat provenance file: %v", err),
		)
	}

	uploadErr := minioClient.UploadFile(ctx, cfg.ArtifactsBucket, provenanceArtifactPath, provFile, provInfo.Size())
	if uploadErr != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			stageLog,
			"provenance_upload",
			fmt.Sprintf("Failed to upload provenance: %v", uploadErr),
		)
	}

	stageLog.recordEvent("provenance_uploaded", map[string]any{"path": provenanceArtifactPath})

	return provenanceArtifactPath
}

func mustStartStaticServer(
	ctx context.Context,
	cfg *Config,
	natsClient *sharedmsg.Client,
	stageLog *stageLogger,
	siteDir string,
) *server.StaticServer {
	slog.Info("Starting static server", "job_id", cfg.JobID, "port", cfg.Port)
	siteServer := server.NewStaticServer(&server.Config{
		SiteDir: siteDir,
		Port:    cfg.Port,
	})

	if err := siteServer.Start(ctx); err != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			stageLog,
			"server",
			fmt.Sprintf("Failed to start static server: %v", err),
		)
	}

	return siteServer
}

func stopStaticServer(ctx context.Context, siteServer *server.StaticServer) {
	if err := siteServer.Stop(ctx); err != nil {
		slog.Error("Failed to stop static server", "error", err)
	}
}

func publishExtractionReady(
	ctx context.Context,
	cfg *Config,
	natsClient *sharedmsg.Client,
	stageLog *stageLogger,
	prov *models.Provenance,
	baseURL, provenancePath, provenanceArtifactPath string,
) {
	slog.Info("Static server started", "job_id", cfg.JobID, "base_url", baseURL)
	stageLog.recordEvent("server_started", map[string]any{"port": cfg.Port})

	slog.Info("Publishing extraction.ready event", "job_id", cfg.JobID)
	payload := &events.ExtractionReadyPayload{
		JobID:                  cfg.JobID,
		ProvenancePath:         provenancePath,
		BaseURL:                baseURL,
		TotalPages:             len(prov.Pages),
		ProvenanceArtifactPath: provenanceArtifactPath,
	}

	stageLog.recordEvent("extraction_ready", map[string]any{"pages": len(prov.Pages)})

	stageLogPath, err := stageLog.finalizeSuccess()
	if err != nil {
		slog.Warn("Failed to finalize extraction stage log", "error", err)
	}

	recipeObject := stageLog.recipeObject()
	payload.StageLogPath = stageLogPath
	payload.RecipePath = recipeObject

	envelope := events.NewEnvelope(events.EventExtractionReady, cfg.JobID, "extractor", payload)
	envelope.RequestID = cfg.RequestID
	envelope.RunID = cfg.RunID

	if publishErr := natsClient.PublishEvent(ctx, sharedmsg.SubjectExtractionReady, envelope); publishErr != nil {
		publishFailureAndExit(
			ctx,
			cfg,
			natsClient,
			stageLog,
			"extraction_ready_publish",
			fmt.Sprintf("Failed to publish extraction.ready event: %v", publishErr),
		)
	}
}

// Config captures the extractor job runtime env.
type Config struct {
	JobID           string
	RequestID       string
	RunID           string
	InputPath       string
	NATS            config.NATSConfig
	MinIO           config.MinIOConfig
	Workspace       string
	Port            string
	ArtifactsBucket string
}

// Validate ensures required env is present before extraction starts.
func (c *Config) Validate() error {
	if c.JobID == "" {
		return errors.New("JOB_ID is required")
	}

	if c.InputPath == "" {
		return errors.New("INPUT_PATH is required")
	}

	if c.NATS.URL == "" {
		return errors.New("NATS_URL is required")
	}

	if c.MinIO.Endpoint == "" {
		return errors.New("MINIO_ENDPOINT is required")
	}

	if c.MinIO.AccessKey == "" {
		return errors.New("MINIO_ACCESS_KEY (or MINIO_ROOT_USER) is required")
	}

	if c.MinIO.SecretKey == "" {
		return errors.New("MINIO_SECRET_KEY (or MINIO_ROOT_PASSWORD) is required")
	}

	if c.Workspace == "" {
		return errors.New("WORKSPACE is required")
	}

	if c.ArtifactsBucket == "" {
		return errors.New("MINIO_ARTIFACT_BUCKET is required")
	}

	return nil
}

func loadConfig() *Config {
	return &Config{
		JobID:           config.GetEnv("JOB_ID", ""),
		RequestID:       config.GetEnv("REQUEST_ID", ""),
		RunID:           config.GetEnv("RUN_ID", ""),
		InputPath:       config.GetEnv("INPUT_PATH", ""),
		NATS:            config.LoadNATSConfig(),
		MinIO:           config.LoadMinIOConfig(),
		Workspace:       config.GetEnv("WORKSPACE", "/workspace"),
		Port:            config.GetEnv("PORT", "8080"),
		ArtifactsBucket: config.GetEnv("MINIO_ARTIFACT_BUCKET", storage.BucketArtifacts),
	}
}

func waitForShutdown(
	ctx context.Context,
	cfg *Config,
	siteServer *server.StaticServer,
	natsClient *sharedmsg.Client,
	stageLog *stageLogger,
) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	defer signal.Stop(stop)

	serverDone := make(chan error, 1)

	go func() {
		serverDone <- siteServer.Wait()
	}()

	select {
	case sig := <-stop:
		slog.Info("Received signal, shutting down", "job_id", cfg.JobID, "signal", sig)
	case err := <-serverDone:
		if err != nil {
			slog.Error("Static server exited with error", "job_id", cfg.JobID, "error", err)
			publishFailureAndExit(
				ctx,
				cfg,
				natsClient,
				stageLog,
				"server",
				fmt.Sprintf("static server crashed: %v", err),
			)
		}

		slog.Info("Static server exited normally", "job_id", cfg.JobID)
	}
}

// publishFailureAndExit centralizes failure reporting so upstream services always get a terminal event.
func publishFailureAndExit(
	ctx context.Context,
	cfg *Config,
	natsClient *sharedmsg.Client,
	stageLog *stageLogger,
	stage, errorMsg string,
) {
	jobID := ""
	requestID := ""
	runID := ""

	if cfg != nil {
		jobID = cfg.JobID
		requestID = cfg.RequestID
		runID = cfg.RunID
	}

	slog.Error("Job FAILED", "job_id", jobID, "stage", stage, "error_message", errorMsg)

	stageLogPath := ""
	recipePath := ""

	if stageLog != nil {
		var err error

		stageLogPath, err = stageLog.finalizeFailure(stage, errorMsg, errorMsg)
		if err != nil {
			slog.Warn("Failed to finalize extraction stage log", "error", err)
		}

		recipePath = stageLog.recipeObject()
	}

	if natsClient != nil {
		payload := &events.ExtractionFailedPayload{
			JobID:        jobID,
			Error:        stage + " failed",
			ErrorDetails: errorMsg,
			StageLogPath: stageLogPath,
			RecipePath:   recipePath,
		}

		envelope := events.NewEnvelope(events.EventExtractionFailed, jobID, "extractor", payload)
		envelope.RequestID = requestID
		envelope.RunID = runID

		if err := natsClient.PublishEvent(ctx, sharedmsg.SubjectExtractionFailed, envelope); err != nil {
			slog.Warn("Failed to publish extraction.failed event", "error", err)
		}
	}

	os.Exit(1)
}
