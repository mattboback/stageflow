package apiclient

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/mattboback/stageflow/libs/go/diff"
)

type RemoteProject struct {
	ID            string    `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	URLs          []string  `json:"urls"`
	Scanners      []string  `json:"scanners,omitempty"`
	BaselineJobID string    `json:"baseline_job_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (c *Client) CreateProject(ctx context.Context, slug, name string, urls, scanners []string) (RemoteProject, error) {
	body := map[string]any{
		"slug": slug,
		"name": name,
		"urls": urls,
	}
	if len(scanners) > 0 {
		body["scanners"] = scanners
	}

	var resp RemoteProject
	if err := c.PostJSON(ctx, "/api/v1/projects", body, &resp); err != nil {
		return RemoteProject{}, err
	}

	return resp, nil
}

func (c *Client) ListProjects(ctx context.Context) ([]RemoteProject, error) {
	var resp []RemoteProject
	if err := c.GetJSON(ctx, "/api/v1/projects", &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) GetProject(ctx context.Context, slug string) (RemoteProject, error) {
	path := fmt.Sprintf("/api/v1/projects/%s", url.PathEscape(slug))

	var resp RemoteProject
	if err := c.GetJSON(ctx, path, &resp); err != nil {
		return RemoteProject{}, err
	}

	return resp, nil
}

func (c *Client) UpdateProject(ctx context.Context, slug string, body map[string]any) (RemoteProject, error) {
	path := fmt.Sprintf("/api/v1/projects/%s", url.PathEscape(slug))

	var resp RemoteProject
	if err := c.PatchJSON(ctx, path, body, &resp); err != nil {
		return RemoteProject{}, err
	}

	return resp, nil
}

func (c *Client) DeleteProject(ctx context.Context, slug string) error {
	path := fmt.Sprintf("/api/v1/projects/%s", url.PathEscape(slug))

	return c.DeleteJSON(ctx, path)
}

func (c *Client) ProjectScan(ctx context.Context, slug string) (SubmitJobResponse, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/scan", url.PathEscape(slug))

	var resp SubmitJobResponse
	if err := c.PostJSON(ctx, path, nil, &resp); err != nil {
		return SubmitJobResponse{}, err
	}

	return resp, nil
}

func (c *Client) PromoteBaseline(ctx context.Context, slug, jobID string) error {
	path := fmt.Sprintf("/api/v1/projects/%s/promote", url.PathEscape(slug))

	return c.PostJSON(ctx, path, map[string]string{"job_id": jobID}, nil)
}

func (c *Client) FetchJobDiff(ctx context.Context, jobID string) (diff.Result, error) {
	path := fmt.Sprintf("/api/v1/jobs/%s/diff", url.PathEscape(jobID))

	var resp struct {
		diff.Result
		Project struct {
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"project"`
	}

	if err := c.GetJSON(ctx, path, &resp); err != nil {
		return diff.Result{}, err
	}

	return resp.Result, nil
}
