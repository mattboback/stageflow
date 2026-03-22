package main

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/mattboback/stageflow/libs/go/diff"
)

type remoteProject struct {
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	URLs          []string `json:"urls"`
	Scanners      []string `json:"scanners,omitempty"`
	BaselineJobID string   `json:"baseline_job_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (c *Client) createProject(ctx context.Context, slug, name string, urls, scanners []string) (remoteProject, error) {
	body := map[string]any{
		"slug": slug,
		"name": name,
		"urls": urls,
	}
	if len(scanners) > 0 {
		body["scanners"] = scanners
	}

	var resp remoteProject
	if err := c.postJSON(ctx, "/api/v1/projects", body, &resp); err != nil {
		return remoteProject{}, err
	}

	return resp, nil
}

func (c *Client) listProjects(ctx context.Context) ([]remoteProject, error) {
	var resp []remoteProject
	if err := c.getJSON(ctx, "/api/v1/projects", &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) getProject(ctx context.Context, slug string) (remoteProject, error) {
	path := fmt.Sprintf("/api/v1/projects/%s", url.PathEscape(slug))

	var resp remoteProject
	if err := c.getJSON(ctx, path, &resp); err != nil {
		return remoteProject{}, err
	}

	return resp, nil
}

func (c *Client) updateProject(ctx context.Context, slug string, body map[string]any) (remoteProject, error) {
	path := fmt.Sprintf("/api/v1/projects/%s", url.PathEscape(slug))

	var resp remoteProject
	if err := c.patchJSON(ctx, path, body, &resp); err != nil {
		return remoteProject{}, err
	}

	return resp, nil
}

func (c *Client) deleteProject(ctx context.Context, slug string) error {
	path := fmt.Sprintf("/api/v1/projects/%s", url.PathEscape(slug))

	return c.deleteJSON(ctx, path)
}

func (c *Client) projectScan(ctx context.Context, slug string) (SubmitJobResponse, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/scan", url.PathEscape(slug))

	var resp SubmitJobResponse
	if err := c.postJSON(ctx, path, nil, &resp); err != nil {
		return SubmitJobResponse{}, err
	}

	return resp, nil
}

func (c *Client) promoteBaseline(ctx context.Context, slug, jobID string) error {
	path := fmt.Sprintf("/api/v1/projects/%s/promote", url.PathEscape(slug))

	return c.postJSON(ctx, path, map[string]string{"job_id": jobID}, nil)
}

func (c *Client) fetchJobDiff(ctx context.Context, jobID string) (diff.Result, error) {
	path := fmt.Sprintf("/api/v1/jobs/%s/diff", url.PathEscape(jobID))

	var resp struct {
		diff.Result
		Project struct {
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"project"`
	}

	if err := c.getJSON(ctx, path, &resp); err != nil {
		return diff.Result{}, err
	}

	return resp.Result, nil
}
