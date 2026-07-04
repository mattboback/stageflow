package stack

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeStageflowCheckout(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	composeDir := filepath.Join(root, "infra", "compose")
	if err := os.MkdirAll(composeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"podman-compose.yml", "podman-compose.test.yml", "podman-compose.local.yml"} {
		if err := os.WriteFile(filepath.Join(composeDir, name), []byte("services: {}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestFindRoot(t *testing.T) {
	t.Run("finds root from nested dir", func(t *testing.T) {
		root := writeStageflowCheckout(t)
		nested := filepath.Join(root, "clients", "cli")

		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}

		got, err := FindRoot(nested)
		if err != nil {
			t.Fatalf("FindRoot() error = %v", err)
		}

		gotAbs, _ := filepath.EvalSymlinks(got)
		wantAbs, _ := filepath.EvalSymlinks(root)

		if gotAbs != wantAbs {
			t.Fatalf("FindRoot() = %q, want %q", got, root)
		}
	})

	t.Run("errors outside a git repo", func(t *testing.T) {
		dir := t.TempDir()

		if _, err := FindRoot(dir); err == nil {
			t.Fatal("FindRoot() error = nil, want non-nil")
		}
	})

	t.Run("errors when compose files are missing", func(t *testing.T) {
		root := t.TempDir()

		if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := FindRoot(root); err == nil {
			t.Fatal("FindRoot() error = nil, want non-nil")
		}
	})
}

func TestComposeFiles(t *testing.T) {
	files, err := ComposeFiles("/repo", EnvDev)
	if err != nil {
		t.Fatalf("ComposeFiles(dev) error = %v", err)
	}

	want := []string{"/repo/infra/compose/podman-compose.yml", "/repo/infra/compose/podman-compose.test.yml"}
	if len(files) != 2 || files[0] != want[0] || files[1] != want[1] {
		t.Fatalf("ComposeFiles(dev) = %v, want %v", files, want)
	}

	files, err = ComposeFiles("/repo", EnvLocal)
	if err != nil {
		t.Fatalf("ComposeFiles(local) error = %v", err)
	}

	want = []string{"/repo/infra/compose/podman-compose.yml", "/repo/infra/compose/podman-compose.local.yml"}
	if len(files) != 2 || files[0] != want[0] || files[1] != want[1] {
		t.Fatalf("ComposeFiles(local) = %v, want %v", files, want)
	}

	if _, err = ComposeFiles("/repo", "staging"); err == nil {
		t.Fatal("ComposeFiles(staging) error = nil, want non-nil")
	}
}

func TestCheckProtectedHost(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		hostname string
		hostErr  error
		wantErr  bool
	}{
		{
			name: "no protected host configured",
			env:  map[string]string{},
		},
		{
			name:     "protected host matches, no override",
			env:      map[string]string{"STAGEFLOW_PROTECTED_HOST": "vps.example.com"},
			hostname: "vps.example.com",
			wantErr:  true,
		},
		{
			name: "protected host matches, override set",
			env: map[string]string{
				"STAGEFLOW_PROTECTED_HOST":         "vps.example.com",
				"STAGEFLOW_ALLOW_VPS_LOCAL_STACKS": "1",
			},
			hostname: "vps.example.com",
		},
		{
			name:     "protected host configured but does not match",
			env:      map[string]string{"STAGEFLOW_PROTECTED_HOST": "vps.example.com"},
			hostname: "laptop.local",
		},
		{
			name:    "hostname lookup fails",
			env:     map[string]string{"STAGEFLOW_PROTECTED_HOST": "vps.example.com"},
			hostErr: errors.New("boom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			hostname := func() (string, error) { return tt.hostname, tt.hostErr }

			err := CheckProtectedHost(getenv, hostname)

			if tt.wantErr {
				var protectedErr *ProtectedHostError
				if !errors.As(err, &protectedErr) {
					t.Fatalf("CheckProtectedHost() error = %v, want *ProtectedHostError", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("CheckProtectedHost() error = %v, want nil", err)
			}
		})
	}
}

func TestNewOptions(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		opts := NewOptions(func(string) string { return "" })

		if opts.Project != DefaultProject {
			t.Fatalf("Project = %q, want %q", opts.Project, DefaultProject)
		}

		if opts.PodmanBin != DefaultPodman {
			t.Fatalf("PodmanBin = %q, want %q", opts.PodmanBin, DefaultPodman)
		}

		if opts.Env != EnvDev {
			t.Fatalf("Env = %q, want %q", opts.Env, EnvDev)
		}
	})

	t.Run("env overrides", func(t *testing.T) {
		env := map[string]string{"COMPOSE_PROJECT_NAME": "custom_project", "PODMAN": "/usr/bin/podman-remote"}
		opts := NewOptions(func(key string) string { return env[key] })

		if opts.Project != "custom_project" {
			t.Fatalf("Project = %q, want custom_project", opts.Project)
		}

		if opts.PodmanBin != "/usr/bin/podman-remote" {
			t.Fatalf("PodmanBin = %q, want /usr/bin/podman-remote", opts.PodmanBin)
		}
	})
}
