package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func maybeTeeOutput(stdout io.Writer, outPath string) (io.Writer, func(), error) {
	if outPath == "" {
		return stdout, func() {}, nil
	}

	dir := filepath.Dir(outPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, func() {}, fmt.Errorf("failed to create output directory %s: %w", dir, err)
		}
	}

	f, err := os.Create(outPath)
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to create output file %s: %w", outPath, err)
	}

	return io.MultiWriter(stdout, f), func() { _ = f.Close() }, nil
}
