package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/storage"
)

const (
	stageSchemaExtraction  = "stageflow.stages.extraction.v1"
	recipeSchemaExtraction = "stageflow.recipes.extraction.v1"
	stageStatusFailed      = "failed"
)

type extractionStageEvent struct {
	Timestamp string         `json:"ts"`
	Type      string         `json:"type"`
	Details   map[string]any `json:"details,omitempty"`
}

type extractionStageMetrics struct {
	PagesDiscovered int `json:"pages_discovered"`
}

type extractionStageArtifacts struct {
	ProvenancePath string `json:"provenance_path,omitempty"`
	BaseURL        string `json:"base_url,omitempty"`
}

type extractionRecipe struct {
	Schema      string                `json:"schema"`
	JobID       string                `json:"job_id"`
	Stage       string                `json:"stage"`
	GeneratedAt string                `json:"generated_at"`
	Input       extractionRecipeInput `json:"input"`
	Workspace   string                `json:"workspace"`
	Port        string                `json:"port"`
	Environment map[string]any        `json:"environment"`
}

type extractionRecipeInput struct {
	Bucket     string `json:"bucket"`
	ObjectPath string `json:"object_path"`
}

type extractionStageLog struct {
	Schema      string                   `json:"schema"`
	JobID       string                   `json:"job_id"`
	Stage       string                   `json:"stage"`
	Attempt     int                      `json:"attempt"`
	Status      string                   `json:"status"`
	StartedAt   string                   `json:"started_at"`
	CompletedAt string                   `json:"completed_at"`
	DurationMs  int64                    `json:"duration_ms"`
	RecipeRef   extractionStageRef       `json:"recipe_ref"`
	Metrics     extractionStageMetrics   `json:"metrics"`
	Events      []extractionStageEvent   `json:"events"`
	Artifacts   extractionStageArtifacts `json:"artifacts"`
	Failure     *extractionStageFailure  `json:"failure,omitempty"`
}

type extractionStageRef struct {
	Bucket string `json:"bucket"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
}

type extractionStageFailure struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type stageLogger struct {
	ctx              context.Context
	jobID            string
	bucket           string
	minioClient      *storage.MinIOClient
	recipePath       string
	recipeObjectPath string
	recipeHash       string
	stageLogPath     string
	stageLogObject   string
	startedAt        time.Time
	events           []extractionStageEvent
	metrics          extractionStageMetrics
	artifacts        extractionStageArtifacts
	mu               sync.Mutex
	finalized        bool
	lastStatus       string
}

func newStageLogger(ctx context.Context, cfg *Config, minioClient *storage.MinIOClient) (*stageLogger, error) {
	recipesDir := filepath.Join(cfg.Workspace, "recipes")
	stagesDir := filepath.Join(cfg.Workspace, "stages")

	if err := os.MkdirAll(recipesDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create recipes directory: %w", err)
	}

	if err := os.MkdirAll(stagesDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create stages directory: %w", err)
	}

	l := &stageLogger{
		ctx:              ctx,
		jobID:            cfg.JobID,
		bucket:           cfg.ArtifactsBucket,
		minioClient:      minioClient,
		recipePath:       filepath.Join(recipesDir, "extraction.json"),
		recipeObjectPath: cfg.JobID + "/recipes/extraction.json",
		stageLogPath:     filepath.Join(stagesDir, "extraction.log.json"),
		stageLogObject:   cfg.JobID + "/stages/extraction.log.json",
		startedAt:        time.Now().UTC(),
	}

	if err := l.writeRecipe(cfg); err != nil {
		return nil, err
	}

	l.events = append(l.events, extractionStageEvent{
		Timestamp: l.startedAt.Format(time.RFC3339Nano),
		Type:      "stage_started",
	})

	return l, nil
}

func (l *stageLogger) writeRecipe(cfg *Config) error {
	recipe := extractionRecipe{
		Schema:      recipeSchemaExtraction,
		JobID:       cfg.JobID,
		Stage:       "extraction",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Input: extractionRecipeInput{
			Bucket:     storage.BucketStaging,
			ObjectPath: cfg.InputPath,
		},
		Workspace: cfg.Workspace,
		Port:      cfg.Port,
		Environment: map[string]any{
			"nats_url":         cfg.NATS.URL,
			"minio_endpoint":   cfg.MinIO.Endpoint,
			"minio_use_ssl":    cfg.MinIO.UseSSL,
			"artifacts_bucket": cfg.ArtifactsBucket,
		},
	}

	data, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal extraction recipe: %w", err)
	}

	if err := os.WriteFile(l.recipePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write extraction recipe: %w", err)
	}

	sum := sha256.Sum256(data)

	l.recipeHash = hex.EncodeToString(sum[:])
	if err := l.uploadBytes(l.recipeObjectPath, data); err != nil {
		return fmt.Errorf("failed to upload extraction recipe: %w", err)
	}

	return nil
}

func (l *stageLogger) uploadBytes(objectPath string, data []byte) error {
	reader := bytes.NewReader(data)
	if err := l.minioClient.UploadFile(l.ctx, l.bucket, objectPath, reader, int64(len(data))); err != nil {
		return err
	}

	return nil
}

func (l *stageLogger) recordEvent(eventType string, details map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, extractionStageEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Type:      eventType,
		Details:   details,
	})
}

func (l *stageLogger) setPagesDiscovered(pages int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.metrics.PagesDiscovered = pages
}

func (l *stageLogger) setArtifacts(provenancePath, baseURL string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.artifacts.ProvenancePath = provenancePath
	l.artifacts.BaseURL = baseURL
}

func (l *stageLogger) finalize(status string, failure *extractionStageFailure) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.finalized {
		if status == stageStatusFailed && l.lastStatus != stageStatusFailed {
			l.finalized = false
		} else {
			return l.stageLogObject, nil
		}
	}

	completed := time.Now().UTC()

	finalEventType := "stage_completed"
	if status == stageStatusFailed {
		finalEventType = "stage_failed"
	}

	if len(l.events) > 0 {
		lastType := l.events[len(l.events)-1].Type
		if lastType == "stage_completed" || lastType == "stage_failed" {
			l.events = l.events[:len(l.events)-1]
		}
	}

	finalEvent := extractionStageEvent{
		Timestamp: completed.Format(time.RFC3339Nano),
		Type:      finalEventType,
	}
	if failure != nil && failure.Message != "" {
		finalEvent.Details = map[string]any{"message": failure.Message}
	}

	l.events = append(l.events, finalEvent)

	log := extractionStageLog{
		Schema:      stageSchemaExtraction,
		JobID:       l.jobID,
		Stage:       "extraction",
		Attempt:     1,
		Status:      status,
		StartedAt:   l.startedAt.Format(time.RFC3339Nano),
		CompletedAt: completed.Format(time.RFC3339Nano),
		DurationMs:  completed.Sub(l.startedAt).Milliseconds(),
		RecipeRef: extractionStageRef{
			Bucket: l.bucket,
			Path:   l.recipeObjectPath,
			SHA256: l.recipeHash,
		},
		Metrics:   l.metrics,
		Events:    l.events,
		Artifacts: l.artifacts,
		Failure:   failure,
	}

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal extraction stage log: %w", err)
	}

	if err := os.WriteFile(l.stageLogPath, data, 0o600); err != nil {
		return "", fmt.Errorf("failed to write extraction stage log: %w", err)
	}

	if err := l.uploadBytes(l.stageLogObject, data); err != nil {
		return "", fmt.Errorf("failed to upload extraction stage log: %w", err)
	}

	l.finalized = true
	l.lastStatus = status

	return l.stageLogObject, nil
}

func (l *stageLogger) finalizeSuccess() (string, error) {
	return l.finalize("succeeded", nil)
}

func (l *stageLogger) finalizeFailure(stage, message, details string) (string, error) {
	failure := &extractionStageFailure{Stage: stage, Message: message, Details: details}

	return l.finalize("failed", failure)
}

func (l *stageLogger) recipeObject() string {
	return l.recipeObjectPath
}
