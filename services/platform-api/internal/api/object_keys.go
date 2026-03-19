package api

import (
	"path"
	"strings"
)

func normalizeObjectKey(raw string) string {
	key := strings.TrimSpace(raw)
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")

	return key
}

func jobScopedKey(jobID, key string) (string, bool) {
	if jobID == "" {
		return "", false
	}

	normalized := normalizeObjectKey(key)
	if normalized == "" {
		return "", false
	}

	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == ".." || cleaned == jobID {
		return "", false
	}

	prefix := jobID + "/"
	if !strings.HasPrefix(cleaned, prefix) {
		return "", false
	}

	return cleaned, true
}

func jobScopedJoin(jobID string, parts ...string) (string, bool) {
	if jobID == "" {
		return "", false
	}

	normalizedParts := make([]string, 0, len(parts)+1)
	normalizedParts = append(normalizedParts, jobID)

	for _, part := range parts {
		part = normalizeObjectKey(part)
		if part == "" {
			continue
		}

		normalizedParts = append(normalizedParts, part)
	}

	joined := path.Clean(path.Join(normalizedParts...))
	if joined == "." || joined == ".." || joined == jobID {
		return "", false
	}

	prefix := jobID + "/"
	if !strings.HasPrefix(joined, prefix) {
		return "", false
	}

	return joined, true
}
