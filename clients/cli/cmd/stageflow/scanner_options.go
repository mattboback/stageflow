package main

import (
	"errors"
	"strings"
)

func normalizeScannerList(scanners []string) ([]string, error) {
	if len(scanners) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(scanners))
	for _, scanner := range scanners {
		trimmed := strings.TrimSpace(scanner)
		if trimmed == "" {
			return nil, errors.New("scanner list contains an empty module name")
		}

		normalized = append(normalized, trimmed)
	}

	return normalized, nil
}
