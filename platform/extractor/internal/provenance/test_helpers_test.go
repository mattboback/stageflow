package provenance

import (
	"os"
	"strings"
	"testing"
)

func mustMkdir(t *testing.T, path string, perm os.FileMode) {
	t.Helper()
	if err := os.Mkdir(path, perm); err != nil {
		t.Fatalf("failed to mkdir %s: %v", path, err)
	}
}

func mustMkdirAll(t *testing.T, path string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, perm); err != nil {
		t.Fatalf("failed to mkdirall %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func isPrettyPrintedJSON(content string) bool {
	if len(content) < 10 {
		return false
	}

	// MarshalIndent should look like:
	// {
	//   "key": ...
	// }
	return strings.HasPrefix(content, "{\n") && strings.Contains(content, "\n  \"")
}
