// Package artifacts stores job inputs and results as files under one data directory.
package artifacts

import (
	"context"
	"io"
)

// Uploader abstracts object writes for services and tests.
type Uploader interface {
	UploadFile(ctx context.Context, bucket, path string, reader io.Reader, size int64) error
}

// Downloader abstracts object reads for services and tests.
type Downloader interface {
	DownloadFile(ctx context.Context, bucket, path string) (io.ReadCloser, error)
}

// Deleter abstracts object cleanup for stages.
type Deleter interface {
	DeleteFile(ctx context.Context, bucket, path string) error
	DeletePrefix(ctx context.Context, bucket, prefix string) error
}

// Client groups the storage surface used by the platform.
type Client interface {
	Uploader
	Downloader
	Deleter
	FileExists(ctx context.Context, bucket, path string) (bool, error)
}
