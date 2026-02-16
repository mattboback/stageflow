package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/mattboback/stageflow/packages/shared-go/config"
)

type minioObject interface {
	io.ReadCloser
	Stat() (minio.ObjectInfo, error)
}

type minioAPI interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	PutObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		reader io.Reader,
		objectSize int64,
		opts minio.PutObjectOptions,
	) (minio.UploadInfo, error)
	PresignedGetObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		expires time.Duration,
		reqParams url.Values,
	) (*url.URL, error)
	GetObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		opts minio.GetObjectOptions,
	) (minioObject, error)
	RemoveObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		opts minio.RemoveObjectOptions,
	) error
	StatObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		opts minio.StatObjectOptions,
	) (minio.ObjectInfo, error)
}

type minioClientAdapter struct {
	client *minio.Client
}

func (a *minioClientAdapter) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	return a.client.BucketExists(ctx, bucketName)
}

func (a *minioClientAdapter) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	return a.client.MakeBucket(ctx, bucketName, opts)
}

func (a *minioClientAdapter) PutObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	reader io.Reader,
	objectSize int64,
	opts minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	return a.client.PutObject(ctx, bucketName, objectName, reader, objectSize, opts)
}

func (a *minioClientAdapter) PresignedGetObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	expires time.Duration,
	reqParams url.Values,
) (*url.URL, error) {
	return a.client.PresignedGetObject(ctx, bucketName, objectName, expires, reqParams)
}

func (a *minioClientAdapter) GetObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	opts minio.GetObjectOptions,
) (minioObject, error) {
	obj, err := a.client.GetObject(ctx, bucketName, objectName, opts)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (a *minioClientAdapter) RemoveObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	opts minio.RemoveObjectOptions,
) error {
	return a.client.RemoveObject(ctx, bucketName, objectName, opts)
}

func (a *minioClientAdapter) StatObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	opts minio.StatObjectOptions,
) (minio.ObjectInfo, error) {
	return a.client.StatObject(ctx, bucketName, objectName, opts)
}

// BucketNames are the stable MinIO buckets StageFlow expects.
const (
	BucketStaging   = "scanner-staging"
	BucketArtifacts = "scanner-artifacts"
)

// MinIOClient manages internal and public MinIO clients for artifact I/O and URL generation.
type MinIOClient struct {
	client       minioAPI
	publicClient minioAPI // Cached client for public endpoint (presigned URLs)
	config       *MinIOConfig
}

// MinIOConfig aliases the shared config to avoid duplicate definitions.
type MinIOConfig = config.MinIOConfig

// NewMinIOClient builds clients and caches a public endpoint client when presigning is enabled.
func NewMinIOClient(cfg *MinIOConfig) (*MinIOClient, error) {
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	internalRaw, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	client := &minioClientAdapter{client: internalRaw}

	// Create public client once if public endpoint is configured
	// This avoids creating a new client on every presigned URL request
	var publicClient minioAPI

	if cfg.PublicEndpoint != "" {
		publicRaw, newErr := minio.New(cfg.PublicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.PublicUseSSL,
			Region: region,
		})
		if newErr != nil {
			return nil, fmt.Errorf("failed to create public MinIO client: %w", newErr)
		}

		publicClient = &minioClientAdapter{client: publicRaw}
	}

	return &MinIOClient{
		client:       client,
		publicClient: publicClient,
		config:       cfg,
	}, nil
}

// EnsureBuckets creates required buckets if they don't exist.
func (c *MinIOClient) EnsureBuckets(ctx context.Context) error {
	buckets := []string{BucketStaging, BucketArtifacts}

	for _, bucket := range buckets {
		if err := c.ensureBucket(ctx, bucket); err != nil {
			return err
		}
	}

	return nil
}

func (c *MinIOClient) ensureBucket(ctx context.Context, bucket string) error {
	const (
		maxRetries = 30
		retryDelay = 2 * time.Second
	)

	return c.ensureBucketWithRetry(ctx, bucket, maxRetries, retryDelay)
}

func (c *MinIOClient) ensureBucketWithRetry(
	ctx context.Context,
	bucket string,
	maxRetries int,
	retryDelay time.Duration,
) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		exists, err := c.client.BucketExists(ctx, bucket)
		if err != nil {
			lastErr = err
			log.Printf(
				"Failed to check/create bucket %s (attempt %d/%d): %v. Retrying in %v...",
				bucket,
				attempt,
				maxRetries,
				lastErr,
				retryDelay,
			)

			if retryErr := waitForRetry(ctx, retryDelay); retryErr != nil {
				return retryErr
			}

			continue
		}

		if exists {
			return nil
		}

		if makeBucketErr := c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); makeBucketErr != nil {
			lastErr = makeBucketErr
			log.Printf(
				"Failed to check/create bucket %s (attempt %d/%d): %v. Retrying in %v...",
				bucket,
				attempt,
				maxRetries,
				lastErr,
				retryDelay,
			)

			if retryErr := waitForRetry(ctx, retryDelay); retryErr != nil {
				return retryErr
			}

			continue
		}

		log.Printf("Created MinIO bucket: %s", bucket)

		return nil
	}

	if lastErr == nil {
		lastErr = errors.New("unknown error ensuring bucket")
	}

	return fmt.Errorf("failed to ensure bucket %s after %d attempts: %w", bucket, maxRetries, lastErr)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// UploadFile uploads to MinIO, using streaming when size is unknown.
func (c *MinIOClient) UploadFile(ctx context.Context, bucket, path string, reader io.Reader, size int64) error {
	objectSize := size
	if objectSize < 0 {
		objectSize = -1 // enable streaming uploads when size unknown
	}

	info, err := c.client.PutObject(ctx, bucket, path, reader, objectSize, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to %s/%s: %w", bucket, path, err)
	}

	log.Printf("Uploaded file to MinIO: %s/%s (%d bytes)", bucket, path, info.Size)

	return nil
}

// GetPresignedURL generates a presigned URL for downloading a file.
// If UseProxyURLs is set, returns a simple public URL for Caddy proxy instead of a presigned URL.
func (c *MinIOClient) GetPresignedURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) {
	if c.config.UseProxyURLs {
		return c.buildPublicProxyURL(bucket, path), nil
	}

	// Use cached public client if available (created once in NewMinIOClient)
	clientToUse := c.client
	if c.publicClient != nil {
		clientToUse = c.publicClient
	}

	url, err := clientToUse.PresignedGetObject(ctx, bucket, path, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s/%s: %w", bucket, path, err)
	}

	return url.String(), nil
}

// buildPublicProxyURL builds a simple public URL for Caddy proxy access.
// Caddy is configured to proxy /{bucket}/* to MinIO.
func (c *MinIOClient) buildPublicProxyURL(bucket, path string) string {
	scheme := "http://"
	if c.config.PublicUseSSL {
		scheme = "https://"
	}

	endpoint := c.config.PublicEndpoint
	if endpoint == "" {
		endpoint = c.config.Endpoint
	}

	return fmt.Sprintf("%s%s/%s/%s", scheme, endpoint, bucket, path)
}

// DownloadFile retrieves an object reader from MinIO.
func (c *MinIOClient) DownloadFile(ctx context.Context, bucket, path string) (io.ReadCloser, error) {
	obj, err := c.client.GetObject(ctx, bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file from %s/%s: %w", bucket, path, err)
	}

	// MinIO may defer "NoSuchKey" until the first read; force a stat so callers get
	// a deterministic error before we hand out a reader.
	if _, statErr := obj.Stat(); statErr != nil {
		_ = obj.Close()

		return nil, fmt.Errorf("failed to download file from %s/%s: %w", bucket, path, statErr)
	}

	return obj, nil
}

// DeleteFile removes an object from MinIO.
func (c *MinIOClient) DeleteFile(ctx context.Context, bucket, path string) error {
	err := c.client.RemoveObject(ctx, bucket, path, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file %s/%s: %w", bucket, path, err)
	}

	log.Printf("Deleted file from MinIO: %s/%s", bucket, path)

	return nil
}

// FileExists returns false on NoSuchKey and surfaces other errors.
func (c *MinIOClient) FileExists(ctx context.Context, bucket, path string) (bool, error) {
	_, err := c.client.StatObject(ctx, bucket, path, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}

		return false, fmt.Errorf("failed to check if file exists %s/%s: %w", bucket, path, err)
	}

	return true, nil
}
