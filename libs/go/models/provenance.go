package models

import (
	"errors"
	"fmt"
	"strings"
)

type Provenance struct {
	Version string `json:"version"`
	JobID   string `json:"job_id"`
	BaseURL string `json:"base_url"`
	Pages   []Page `json:"pages"`
}

func (p *Provenance) Validate() error {
	if p == nil {
		return errors.New("models: Provenance is nil")
	}

	if p.Version == "" {
		return errors.New("models: Provenance.version is required")
	}

	if p.JobID == "" {
		return errors.New("models: Provenance.job_id is required")
	}

	if p.BaseURL == "" {
		return errors.New("models: Provenance.base_url is required")
	}

	if len(p.Pages) == 0 {
		return errors.New("models: Provenance.pages must not be empty")
	}

	for i := range p.Pages {
		if err := p.Pages[i].Validate(); err != nil {
			return fmt.Errorf("models: Provenance.pages[%d] invalid: %w", i, err)
		}
	}

	return nil
}

type Page struct {
	ID   string `json:"id"`
	Path string `json:"path"` // Relative path, e.g., "/index.html"
	URL  string `json:"url"`  // Full URL, e.g., "http://localhost:8080/index.html"
}

func (p Page) Validate() error {
	if p.ID == "" {
		return errors.New("models: Page.id is required")
	}

	if p.Path == "" {
		return errors.New("models: Page.path is required")
	}

	if !strings.HasPrefix(p.Path, "/") {
		return fmt.Errorf("models: Page.path must start with '/' (got %q)", p.Path)
	}

	if p.URL == "" {
		return errors.New("models: Page.url is required")
	}

	return nil
}
