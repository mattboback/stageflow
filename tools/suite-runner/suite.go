package main

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func loadSuite(path string) (Suite, error) {
	//nolint:gosec // Intended behavior for CLI tool to read user-provided file
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Suite{}, err
	}

	var s Suite
	if err := yaml.Unmarshal(data, &s); err != nil {
		return Suite{}, err
	}

	if len(s.Domains) == 0 {
		return Suite{}, errors.New("suite must specify at least one domain")
	}

	return s, nil
}

func applyDefaults(s *Suite) {
	if len(s.Modules) == 0 {
		s.Modules = []string{"axe", "keyboard"}
	}

	if s.TimeoutSec == 0 {
		s.TimeoutSec = 900
	}

	if s.StreamRetry == 0 {
		s.StreamRetry = 3
	}
}
