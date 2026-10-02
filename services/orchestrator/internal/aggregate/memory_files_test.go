package aggregate

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// memoryFiles keeps job files in memory, keyed "<job ID>/<path>".
type memoryFiles struct {
	files map[string][]byte
}

func newMemoryFiles() *memoryFiles {
	return &memoryFiles{files: make(map[string][]byte)}
}

func (m *memoryFiles) Write(_ context.Context, key string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	m.files[key] = data

	return nil
}

func (m *memoryFiles) Open(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := m.files[key]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", key)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}
