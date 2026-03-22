package project

import (
	"context"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return s
}

func TestCreateAndGetProject(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p := &Project{
		Slug: "my-site",
		Name: "My Site",
		URLs: []string{"https://example.com"},
	}

	if err := s.CreateProject(ctx, p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	if p.ID == "" {
		t.Fatal("expected ID to be set")
	}

	got, err := s.GetProjectBySlug(ctx, "my-site")
	if err != nil {
		t.Fatalf("GetProjectBySlug: %v", err)
	}

	if got.Slug != "my-site" || got.Name != "My Site" {
		t.Errorf("slug=%q name=%q", got.Slug, got.Name)
	}

	if len(got.URLs) != 1 || got.URLs[0] != "https://example.com" {
		t.Errorf("urls=%v", got.URLs)
	}

	gotByID, err := s.GetProjectByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetProjectByID: %v", err)
	}

	if gotByID.Slug != "my-site" {
		t.Errorf("GetProjectByID slug=%q", gotByID.Slug)
	}
}

func TestCreateDuplicateSlug(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p := &Project{Slug: "dup", Name: "Dup", URLs: []string{"https://a.com"}}
	if err := s.CreateProject(ctx, p); err != nil {
		t.Fatalf("first create: %v", err)
	}

	p2 := &Project{Slug: "dup", Name: "Dup2", URLs: []string{"https://b.com"}}
	if err := s.CreateProject(ctx, p2); err == nil {
		t.Fatal("expected error on duplicate slug")
	}
}

func TestListProjects(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects (empty): %v", err)
	}

	if len(projects) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(projects))
	}

	_ = s.CreateProject(ctx, &Project{Slug: "aa", Name: "AA", URLs: []string{"https://a.com"}})
	_ = s.CreateProject(ctx, &Project{Slug: "bb", Name: "BB", URLs: []string{"https://b.com"}})

	projects, err = s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
}

func TestUpdateProject(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p := &Project{Slug: "up", Name: "Before", URLs: []string{"https://old.com"}}
	_ = s.CreateProject(ctx, p)

	newName := "After"
	if err := s.UpdateProject(ctx, p.ID, Update{
		Name: &newName,
		URLs: []string{"https://new.com", "https://new2.com"},
	}); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	got, _ := s.GetProjectByID(ctx, p.ID)
	if got.Name != "After" {
		t.Errorf("name=%q", got.Name)
	}

	if len(got.URLs) != 2 {
		t.Errorf("urls=%v", got.URLs)
	}
}

func TestDeleteProject(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p := &Project{Slug: "del", Name: "Del", URLs: []string{"https://del.com"}}
	_ = s.CreateProject(ctx, p)

	if err := s.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	_, err := s.GetProjectBySlug(ctx, "del")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.DeleteProject(ctx, "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProjectJobMapping(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p := &Project{Slug: "mapped", Name: "Mapped", URLs: []string{"https://mapped.com"}}
	_ = s.CreateProject(ctx, p)

	jobID := "job-123"
	if err := s.RecordProjectJob(ctx, p.ID, jobID); err != nil {
		t.Fatalf("RecordProjectJob: %v", err)
	}

	got, err := s.GetProjectForJob(ctx, jobID)
	if err != nil {
		t.Fatalf("GetProjectForJob: %v", err)
	}

	if got.Slug != "mapped" {
		t.Errorf("slug=%q", got.Slug)
	}

	belongs, err := s.JobBelongsToProject(ctx, p.ID, jobID)
	if err != nil {
		t.Fatalf("JobBelongsToProject: %v", err)
	}

	if !belongs {
		t.Error("expected job to belong to project")
	}

	belongs, err = s.JobBelongsToProject(ctx, p.ID, "other-job")
	if err != nil {
		t.Fatalf("JobBelongsToProject (other): %v", err)
	}

	if belongs {
		t.Error("expected other job NOT to belong to project")
	}
}

func TestGetProjectForJobNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.GetProjectForJob(ctx, "unknown-job")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSetBaseline(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p := &Project{Slug: "bl", Name: "BL", URLs: []string{"https://bl.com"}}
	_ = s.CreateProject(ctx, p)
	_ = s.RecordProjectJob(ctx, p.ID, "job-base")

	if err := s.SetBaseline(ctx, p.ID, "job-base"); err != nil {
		t.Fatalf("SetBaseline: %v", err)
	}

	got, _ := s.GetProjectByID(ctx, p.ID)
	if got.BaselineJobID != "job-base" {
		t.Errorf("baseline=%q", got.BaselineJobID)
	}
}

func TestDeleteProjectCascadesJobs(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p := &Project{Slug: "cascade", Name: "Cascade", URLs: []string{"https://cascade.com"}}
	_ = s.CreateProject(ctx, p)
	_ = s.RecordProjectJob(ctx, p.ID, "job-cas")

	_ = s.DeleteProject(ctx, p.ID)

	_, err := s.GetProjectForJob(ctx, "job-cas")
	if err != ErrNotFound {
		t.Fatalf("expected cascade delete, got %v", err)
	}
}

func TestScanners(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p := &Project{
		Slug:     "sc",
		Name:     "SC",
		URLs:     []string{"https://sc.com"},
		Scanners: []string{"axe", "seo"},
	}
	_ = s.CreateProject(ctx, p)

	got, _ := s.GetProjectBySlug(ctx, "sc")
	if len(got.Scanners) != 2 || got.Scanners[0] != "axe" || got.Scanners[1] != "seo" {
		t.Errorf("scanners=%v", got.Scanners)
	}

	// Update scanners to empty (clear them)
	if err := s.UpdateProject(ctx, p.ID, Update{Scanners: []string{}}); err != nil {
		t.Fatalf("UpdateProject scanners: %v", err)
	}

	got, _ = s.GetProjectByID(ctx, p.ID)
	if len(got.Scanners) != 0 {
		t.Errorf("expected empty scanners after clear, got %v", got.Scanners)
	}
}
