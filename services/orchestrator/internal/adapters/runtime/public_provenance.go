package podman

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/libs/go/models"
)

func (r *JobRuntime) publishURLProvenance(ctx context.Context, job *models.Job) error {
	if job.InputType != models.JobInputTypeURLs {
		return nil
	}

	files, err := artifacts.NewFiles(r.dataDir)
	if err != nil {
		return err
	}
	defer files.Close()

	origins := map[string]bool{}

	for _, raw := range job.URLs {
		if parsed, parseErr := url.Parse(raw); parseErr == nil {
			origins[strings.ToLower(parsed.Scheme+"://"+parsed.Host)] = true
		}
	}

	includeOrigin := len(origins) > 1 || job.Config.IdentityScope == "origin"

	pages := make([]map[string]string, 0, len(job.URLs))
	for _, raw := range job.URLs {
		parsed, parseErr := url.Parse(raw)
		if parseErr != nil {
			return parseErr
		}

		identity := parsed.EscapedPath()
		if identity == "" {
			identity = "/"
		}

		if parsed.RawQuery != "" {
			identity += "?" + parsed.RawQuery
		}

		if includeOrigin {
			identity = strings.ToLower(parsed.Scheme+"://"+parsed.Host) + identity
		}

		hash := sha256.Sum256([]byte(identity))
		pages = append(
			pages,
			map[string]string{"id": "page-v2-" + hex.EncodeToString(hash[:8]), "path": "", "url": raw},
		)
	}

	doc := map[string]any{
		"version":  "1.0.0",
		"job_id":   job.ID,
		"base_url": "",
		"mode":     "live",
		"pages":    pages,
		"metadata": map[string]any{"auth_configured": len(job.Config.Auth) > 0},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	return files.UploadFile(
		ctx,
		artifacts.BucketArtifacts,
		job.ID+"/provenance.json",
		bytes.NewReader(data),
		int64(len(data)),
	)
}
