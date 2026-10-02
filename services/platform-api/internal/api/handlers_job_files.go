package api

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mattboback/stageflow/internal/jobstore"

	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
)

// jobRelative returns key's path inside the job's directory.
func jobRelative(jobID, key string) (string, bool) {
	if jobID == "" {
		return "", false
	}

	return strings.CutPrefix(key, jobID+"/")
}

// publicArtifactURL returns the API route that serves one of a job's public files.
func publicArtifactURL(jobID, key string) (string, error) {
	relative, ok := jobRelative(jobID, key)
	if !ok {
		return "", errors.New("artifact is not job scoped")
	}

	if !artifacts.PublicPath(relative) {
		return "", errors.New("artifact is private")
	}

	segments := strings.Split(relative, "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}

	return "/api/v1/jobs/" + url.PathEscape(jobID) + "/files/" + strings.Join(segments, "/"), nil
}

func (s *Server) handleJobFile(w http.ResponseWriter, r *http.Request, jobID, relative string) {
	if !artifacts.PublicPath(relative) {
		httputil.RespondNotFound(w, "File")
		return
	}

	job, readErr := s.statusReader.GetJob(r.Context(), jobID)
	if readErr != nil || job == nil {
		if errors.Is(readErr, jobstore.ErrJobNotFound) || (readErr == nil && job == nil) {
			httputil.RespondNotFound(w, "Job")
		} else {
			httputil.RespondError(w, http.StatusServiceUnavailable, "Failed to read job")
		}

		return
	}

	reader, err := s.config.Files.Open(r.Context(), jobID+"/"+relative)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			httputil.RespondNotFound(w, "File")
		} else {
			httputil.RespondError(w, 500, "Failed to read file")
		}

		return
	}
	defer reader.Close()

	w.Header().Set("X-Content-Type-Options", "nosniff")

	switch {
	case strings.HasSuffix(relative, ".json"):
		w.Header().Set("Content-Type", "application/json")
	case strings.HasSuffix(relative, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(relative, ".jpg"), strings.HasSuffix(relative, ".jpeg"):
		w.Header().Set("Content-Type", "image/jpeg")
	case strings.HasSuffix(relative, ".webp"):
		w.Header().Set("Content-Type", "image/webp")
	}

	if r.Method == http.MethodHead {
		return
	}

	if _, copyErr := io.Copy(w, io.LimitReader(reader, artifacts.MaxArtifactBytes)); copyErr != nil {
		logging.Warn(r.Context(), "Artifact download interrupted", "error", copyErr)
	}
}
