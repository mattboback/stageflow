package main

import "testing"

func TestScanPolicyConfiguration(t *testing.T) {
	t.Setenv("PLATFORM_API_MAX_PAGES", "")
	t.Setenv("PLATFORM_API_ZIP_UPLOAD_ENABLED", "")

	cfg := loadConfig()
	if cfg.MaxPages != 100 || !cfg.ZIPUploadEnabled {
		t.Fatalf("self-host policy: %+v", cfg)
	}

	t.Setenv("PLATFORM_API_MAX_PAGES", "25")
	t.Setenv("PLATFORM_API_ZIP_UPLOAD_ENABLED", "false")

	cfg = loadConfig()
	if cfg.MaxPages != 25 || cfg.ZIPUploadEnabled {
		t.Fatal("hosted policy was not loaded")
	}

	cfg.DatabaseURL = "postgres://localhost/test"
	for _, max := range []int{0, -1, 101} {
		cfg.MaxPages = max
		if cfg.Validate() == nil {
			t.Fatalf("accepted invalid page limit %d", max)
		}
	}
}
