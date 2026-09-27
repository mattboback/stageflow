package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/stack"
)

func stackCheckout(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	head := []byte("ref: refs/heads/main\n")
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), head, 0o600); err != nil {
		t.Fatal(err)
	}

	composeDir := filepath.Join(root, "infra", "compose")
	if err := os.MkdirAll(composeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	composeYAML := []byte("services: {}\n")
	if err := os.WriteFile(filepath.Join(composeDir, "podman-compose.yml"), composeYAML, 0o600); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestResolveStackOptions(t *testing.T) {
	root := stackCheckout(t)
	t.Chdir(root)

	t.Run("resolves root and defaults", func(t *testing.T) {
		opts, resolveErr := resolveStackOptions(
			func(string) string { return "" },
			&stackCmdOptions{env: stack.EnvDev},
			[]string{"platform-api"},
		)
		if resolveErr != nil {
			t.Fatalf("resolveStackOptions() error = %v", resolveErr)
		}

		gotRoot, _ := filepath.EvalSymlinks(opts.Root)
		wantRoot, _ := filepath.EvalSymlinks(root)

		if gotRoot != wantRoot {
			t.Fatalf("Root = %q, want %q", opts.Root, root)
		}

		if opts.Project != "" {
			t.Fatalf("Project = %q, want unset so Compose can read .env", opts.Project)
		}

		if len(opts.Services) != 1 || opts.Services[0] != "platform-api" {
			t.Fatalf("Services = %v, want [platform-api]", opts.Services)
		}
	})

	t.Run("project flag overrides shell and preserves env file", func(t *testing.T) {
		env := map[string]string{"COMPOSE_PROJECT_NAME": "shell_project", "ENV_FILE": "/tmp/selected.env"}

		opts, resolveErr := resolveStackOptions(
			func(key string) string { return env[key] },
			&stackCmdOptions{env: stack.EnvLocal, project: "custom"},
			nil,
		)
		if resolveErr != nil {
			t.Fatalf("resolveStackOptions() error = %v", resolveErr)
		}

		if opts.Project != "custom" {
			t.Fatalf("Project = %q, want custom", opts.Project)
		}

		if opts.Env != stack.EnvLocal {
			t.Fatalf("Env = %q, want %q", opts.Env, stack.EnvLocal)
		}

		if opts.EnvFile != "/tmp/selected.env" {
			t.Fatalf("EnvFile = %q, want selected external env file", opts.EnvFile)
		}
	})
}

func TestResolveStackOptions_NotAStageflowCheckout(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Chdir(t.TempDir())
	t.Cleanup(func() { _ = os.Chdir(wd) })

	_, resolveErr := resolveStackOptions(func(string) string { return "" }, &stackCmdOptions{env: stack.EnvDev}, nil)
	if resolveErr == nil {
		t.Fatal("resolveStackOptions() error = nil, want non-nil")
	}
}
