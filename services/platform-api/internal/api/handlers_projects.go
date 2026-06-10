package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/project"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

const maxProjectRequestBodySize = 1 * 1024 * 1024 // 1 MiB

func validSlug(s string) bool {
	if len(s) < 2 || len(s) > 64 {
		return false
	}

	return slugPattern.MatchString(s)
}

// handleProjects routes /api/v1/projects and /api/v1/projects/ paths.
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/projects")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleListProjects(w, r)
		case http.MethodPost:
			s.handleCreateProject(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

		return
	}

	parts := strings.SplitN(path, "/", 2)
	slug := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleGetProject(w, r, slug)
		case http.MethodPatch:
			s.handleUpdateProject(w, r, slug)
		case http.MethodDelete:
			s.handleDeleteProject(w, r, slug)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

		return
	}

	switch parts[1] {
	case "scan":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

			return
		}

		s.handleProjectScan(w, r, slug)
	case "promote":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

			return
		}

		s.handleProjectPromote(w, r, slug)
	default:
		httputil.RespondNotFound(w, "Endpoint")
	}
}

type createProjectRequest struct {
	Slug     string   `json:"slug"`
	Name     string   `json:"name"`
	URLs     []string `json:"urls"`
	Scanners []string `json:"scanners,omitempty"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := decodeProjectRequestBody(w, r, &req); err != nil {
		handleProjectRequestDecodeError(w, err)

		return
	}

	if !validSlug(req.Slug) {
		httputil.RespondError(w, http.StatusBadRequest,
			"Slug must be 2-64 characters, lowercase alphanumeric and hyphens, cannot start or end with hyphen")

		return
	}

	if req.Name == "" {
		req.Name = req.Slug
	}

	if !s.validateProjectURLs(w, r, req.URLs) {
		return
	}

	if _, err := s.normalizeModules(req.Scanners); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err.Error())

		return
	}

	p := &project.Project{
		Slug:     req.Slug,
		Name:     req.Name,
		URLs:     req.URLs,
		Scanners: req.Scanners,
	}

	if err := s.projectStore.CreateProject(r.Context(), p); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			httputil.RespondError(w, http.StatusConflict, fmt.Sprintf("Project with slug %q already exists", req.Slug))

			return
		}

		logging.Error(r.Context(), "Failed to create project", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to create project")

		return
	}

	httputil.RespondCreated(w, p)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.projectStore.ListProjects(r.Context())
	if err != nil {
		logging.Error(r.Context(), "Failed to list projects", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to list projects")

		return
	}

	if projects == nil {
		projects = []project.Project{}
	}

	httputil.RespondOK(w, projects)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request, slug string) {
	p, err := s.projectStore.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, project.ErrNotFound) {
			httputil.RespondNotFound(w, "Project")

			return
		}

		logging.Error(r.Context(), "Failed to get project", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to get project")

		return
	}

	httputil.RespondOK(w, p)
}

func (s *Server) validateProjectURLs(w http.ResponseWriter, r *http.Request, urls []string) bool {
	if detail := validateURLSubmitRequest(urls); detail != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, *detail)

		return false
	}

	// Projects carry no per-request opt-in, so the instance-level setting
	// decides whether private project URLs are accepted.
	validationMode := targetValidationModePublic
	if s.config.AllowPrivateTargets {
		validationMode = targetValidationModePrivate
	}

	if err := validateTargetURLsWithResolver(r.Context(), s.ipResolver, urls, validationMode); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err.Error())

		return false
	}

	return true
}

type updateProjectRequest struct {
	Name     *string   `json:"name,omitempty"`
	URLs     *[]string `json:"urls,omitempty"`
	Scanners []string  `json:"scanners,omitempty"`
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request, slug string) {
	p, err := s.projectStore.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, project.ErrNotFound) {
			httputil.RespondNotFound(w, "Project")

			return
		}

		logging.Error(r.Context(), "Failed to get project", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to get project")

		return
	}

	var req updateProjectRequest

	err = decodeProjectRequestBody(w, r, &req)
	if err != nil {
		handleProjectRequestDecodeError(w, err)

		return
	}

	var urls []string

	if req.URLs != nil {
		if !s.validateProjectURLs(w, r, *req.URLs) {
			return
		}

		urls = *req.URLs
	}

	if req.Scanners != nil {
		if _, normalizeErr := s.normalizeModules(req.Scanners); normalizeErr != nil {
			httputil.RespondError(w, http.StatusBadRequest, normalizeErr.Error())

			return
		}
	}

	err = s.projectStore.UpdateProject(r.Context(), p.ID, project.Update{
		Name:     req.Name,
		URLs:     urls,
		Scanners: req.Scanners,
	})
	if err != nil {
		logging.Error(r.Context(), "Failed to update project", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to update project")

		return
	}

	updated, err := s.projectStore.GetProjectByID(r.Context(), p.ID)
	if err != nil {
		logging.Error(r.Context(), "Failed to reload project", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to reload updated project")

		return
	}

	httputil.RespondOK(w, updated)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request, slug string) {
	p, err := s.projectStore.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, project.ErrNotFound) {
			httputil.RespondNotFound(w, "Project")

			return
		}

		logging.Error(r.Context(), "Failed to get project", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to get project")

		return
	}

	err = s.projectStore.DeleteProject(r.Context(), p.ID)
	if err != nil {
		logging.Error(r.Context(), "Failed to delete project", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to delete project")

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleProjectScan(w http.ResponseWriter, r *http.Request, slug string) {
	p, err := s.projectStore.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, project.ErrNotFound) {
			httputil.RespondNotFound(w, "Project")

			return
		}

		logging.Error(r.Context(), "Failed to get project", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to get project")

		return
	}

	modules, err := s.normalizeModules(p.Scanners)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err.Error())

		return
	}

	if !s.validateProjectURLs(w, r, p.URLs) {
		return
	}

	jobID := uuid.New().String()
	ctx := logging.WithJobID(r.Context(), jobID)
	runID := uuid.New().String()[:8]
	ctx = logging.WithRunID(ctx, runID)

	payload := &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "urls",
		URLs:      p.URLs,
		Config: models.JobConfig{
			Modules:    modules,
			Screenshot: defaultScreenshot,
			// Scanner pods enforce their own target guard; carry the
			// instance-level opt-in so private project URLs stay scannable.
			AllowPrivateTargets: s.config.AllowPrivateTargets,
		},
	}

	envelope := events.NewEnvelope(events.EventJobCreated, jobID, "platform-api", payload)
	envelope.RequestID = logging.RequestID(ctx)
	envelope.RunID = logging.RunID(ctx)

	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			httputil.RespondError(w, http.StatusServiceUnavailable, "Request timeout")
		} else {
			httputil.RespondError(w, http.StatusRequestTimeout, "Request canceled")
		}

		return
	}

	if recordErr := s.projectStore.RecordProjectJob(ctx, p.ID, jobID); recordErr != nil {
		logging.Error(ctx, "Failed to record project job mapping", "error", recordErr)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to record project scan")

		return
	}

	if publishErr := s.config.Publisher.PublishJobCreated(ctx, envelope); publishErr != nil {
		if deleteErr := s.projectStore.DeleteProjectJob(context.Background(), jobID); deleteErr != nil {
			logging.Warn(ctx, "Failed to clean up project job mapping after publish failure", "error", deleteErr)
		}

		logging.Error(ctx, "Failed to publish job.created event", "error", publishErr)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to queue scan job")

		return
	}

	if _, beginErr := s.jobStatus.Begin(ctx, jobstatus.BeginJob{
		Payload:    payload,
		ObservedAt: envelope.Timestamp,
	}); beginErr != nil {
		logging.Warn(ctx, "Failed to seed provisional job status", "error", beginErr)
	}

	logging.Info(ctx, "Project scan created", "project", p.Slug, "url_count", len(p.URLs))

	httputil.RespondCreated(w, map[string]any{
		"job_id":  jobID,
		"project": p.Slug,
		"status":  "pending",
		"message": "Scan job created for project",
	})
}

type promoteRequest struct {
	JobID string `json:"job_id"`
}

func (s *Server) handleProjectPromote(w http.ResponseWriter, r *http.Request, slug string) {
	p, err := s.projectStore.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, project.ErrNotFound) {
			httputil.RespondNotFound(w, "Project")

			return
		}

		logging.Error(r.Context(), "Failed to get project", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to get project")

		return
	}

	var req promoteRequest

	err = decodeProjectRequestBody(w, r, &req)
	if err != nil {
		handleProjectRequestDecodeError(w, err)

		return
	}

	if req.JobID == "" {
		httputil.RespondError(w, http.StatusBadRequest, "job_id is required")

		return
	}

	belongs, err := s.projectStore.JobBelongsToProject(r.Context(), p.ID, req.JobID)
	if err != nil {
		logging.Error(r.Context(), "Failed to check job ownership", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to verify job ownership")

		return
	}

	if !belongs {
		httputil.RespondError(w, http.StatusBadRequest,
			fmt.Sprintf("Job %s does not belong to project %q", req.JobID, p.Slug))

		return
	}

	rec, err := s.jobStatus.Current(r.Context(), req.JobID)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, fmt.Sprintf("Job %s not found", req.JobID))

		return
	}

	if rec.State != models.JobStateDone {
		httputil.RespondError(w, http.StatusBadRequest,
			fmt.Sprintf("Job %s is not completed (state: %s)", req.JobID, rec.State))

		return
	}

	err = s.projectStore.SetBaseline(r.Context(), p.ID, req.JobID)
	if err != nil {
		logging.Error(r.Context(), "Failed to set baseline", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to set baseline")

		return
	}

	httputil.RespondOK(w, map[string]any{
		"project":         p.Slug,
		"baseline_job_id": req.JobID,
		"message":         "Baseline updated",
	})
}

func decodeProjectRequestBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxProjectRequestBodySize)
	return json.NewDecoder(r.Body).Decode(dst)
}

func handleProjectRequestDecodeError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		httputil.RespondStructuredError(
			w,
			http.StatusRequestEntityTooLarge,
			httputil.NewPayloadTooLargeError(strconv.Itoa(maxProjectRequestBodySize)),
		)

		return
	}

	msg := err.Error()
	if strings.Contains(msg, "request body too large") {
		httputil.RespondStructuredError(
			w,
			http.StatusRequestEntityTooLarge,
			httputil.NewPayloadTooLargeError(strconv.Itoa(maxProjectRequestBodySize)),
		)

		return
	}

	httputil.RespondError(w, http.StatusBadRequest, "Invalid JSON body")
}
