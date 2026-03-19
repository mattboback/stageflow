package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

type memoryStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{objects: make(map[string][]byte)}
}

func (m *memoryStorage) UploadFile(_ context.Context, bucket, path string, reader io.Reader, _ int64) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.objects[m.key(bucket, path)] = data

	return nil
}

func (m *memoryStorage) DownloadFile(_ context.Context, bucket, path string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, ok := m.objects[m.key(bucket, path)]
	if !ok {
		return nil, fmt.Errorf("object not found: %s/%s", bucket, path)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *memoryStorage) DeleteFile(_ context.Context, bucket, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.objects, m.key(bucket, path))

	return nil
}

func (m *memoryStorage) GetPresignedURL(_ context.Context, bucket, path string, _ time.Duration) (string, error) {
	return fmt.Sprintf("https://storage.local/%s/%s", bucket, path), nil
}

func (m *memoryStorage) FileExists(_ context.Context, bucket, path string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.objects[m.key(bucket, path)]

	return ok, nil
}

func (m *memoryStorage) key(bucket, path string) string {
	return fmt.Sprintf("%s::%s", bucket, path)
}
