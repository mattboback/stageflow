package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

type fakeMinioClient struct {
	bucketExistsBuckets []string
	makeBucketBuckets   []string
	putObjectSizes      []int64
	presignCalls        int
	getObjectCalls      int
	removeObjectCalls   int
	statObjectCalls     int

	BucketExistsFn func(ctx context.Context, bucketName string) (bool, error)
	MakeBucketFn   func(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	PutObjectFn    func(
		ctx context.Context,
		bucketName string,
		objectName string,
		reader *bytes.Reader,
		objectSize int64,
		opts minio.PutObjectOptions,
	) (minio.UploadInfo, error)
	PresignFn func(
		ctx context.Context,
		bucketName string,
		objectName string,
		expires time.Duration,
		reqParams url.Values,
	) (*url.URL, error)
	GetObjectFn func(
		ctx context.Context,
		bucketName string,
		objectName string,
		opts minio.GetObjectOptions,
	) (minioObject, error)
	RemoveObjectFn func(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	StatObjectFn   func(
		ctx context.Context,
		bucketName string,
		objectName string,
		opts minio.StatObjectOptions,
	) (minio.ObjectInfo, error)
}

func (f *fakeMinioClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	f.bucketExistsBuckets = append(f.bucketExistsBuckets, bucketName)
	if f.BucketExistsFn != nil {
		return f.BucketExistsFn(ctx, bucketName)
	}

	return false, nil
}

func (f *fakeMinioClient) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	f.makeBucketBuckets = append(f.makeBucketBuckets, bucketName)
	if f.MakeBucketFn != nil {
		return f.MakeBucketFn(ctx, bucketName, opts)
	}

	return nil
}

func (f *fakeMinioClient) PutObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	reader io.Reader,
	objectSize int64,
	opts minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	f.putObjectSizes = append(f.putObjectSizes, objectSize)

	if r, ok := reader.(*bytes.Reader); ok && f.PutObjectFn != nil {
		return f.PutObjectFn(ctx, bucketName, objectName, r, objectSize, opts)
	}

	if f.PutObjectFn != nil {
		// Ensure tests can still assert size/opts without caring about reader type.
		return f.PutObjectFn(ctx, bucketName, objectName, bytes.NewReader(nil), objectSize, opts)
	}

	return minio.UploadInfo{Size: objectSize}, nil
}

func (f *fakeMinioClient) PresignedGetObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	expires time.Duration,
	reqParams url.Values,
) (*url.URL, error) {
	f.presignCalls++
	if f.PresignFn != nil {
		return f.PresignFn(ctx, bucketName, objectName, expires, reqParams)
	}

	return nil, errors.New("no presign stub")
}

func (f *fakeMinioClient) GetObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	opts minio.GetObjectOptions,
) (minioObject, error) {
	f.getObjectCalls++
	if f.GetObjectFn != nil {
		return f.GetObjectFn(ctx, bucketName, objectName, opts)
	}

	return nil, errors.New("no get object stub")
}

func (f *fakeMinioClient) RemoveObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	opts minio.RemoveObjectOptions,
) error {
	f.removeObjectCalls++
	if f.RemoveObjectFn != nil {
		return f.RemoveObjectFn(ctx, bucketName, objectName, opts)
	}

	return nil
}

func (f *fakeMinioClient) StatObject(
	ctx context.Context,
	bucketName string,
	objectName string,
	opts minio.StatObjectOptions,
) (minio.ObjectInfo, error) {
	f.statObjectCalls++
	if f.StatObjectFn != nil {
		return f.StatObjectFn(ctx, bucketName, objectName, opts)
	}

	return minio.ObjectInfo{}, nil
}

type fakeObject struct {
	reader  *bytes.Reader
	closed  bool
	statErr error
}

func newFakeObject(data []byte, statErr error) *fakeObject {
	return &fakeObject{
		reader:  bytes.NewReader(data),
		statErr: statErr,
	}
}

func (o *fakeObject) Read(p []byte) (int, error) { return o.reader.Read(p) }
func (o *fakeObject) Close() error               { o.closed = true; return nil }
func (o *fakeObject) Stat() (minio.ObjectInfo, error) {
	if o.statErr != nil {
		return minio.ObjectInfo{}, o.statErr
	}

	return minio.ObjectInfo{}, nil
}

func TestEnsureBuckets_DoesNotCreateWhenExists(t *testing.T) {
	fake := &fakeMinioClient{
		BucketExistsFn: func(_ context.Context, _ string) (bool, error) { return true, nil },
		MakeBucketFn: func(_ context.Context, bucketName string, _ minio.MakeBucketOptions) error {
			t.Fatalf("unexpected make bucket call: %s", bucketName)

			return nil
		},
	}

	client := &MinIOClient{client: fake, config: &MinIOConfig{}}
	if err := client.EnsureBuckets(context.Background()); err != nil {
		t.Fatalf("EnsureBuckets: %v", err)
	}

	if got := len(fake.bucketExistsBuckets); got != 2 {
		t.Fatalf("expected 2 BucketExists calls, got %d", got)
	}

	if len(fake.makeBucketBuckets) != 0 {
		t.Fatalf("expected no MakeBucket calls, got %d", len(fake.makeBucketBuckets))
	}
}

func TestEnsureBuckets_CreatesMissingBuckets(t *testing.T) {
	fake := &fakeMinioClient{
		BucketExistsFn: func(_ context.Context, _ string) (bool, error) { return false, nil },
	}

	client := &MinIOClient{client: fake, config: &MinIOConfig{}}
	if err := client.EnsureBuckets(context.Background()); err != nil {
		t.Fatalf("EnsureBuckets: %v", err)
	}

	if got := len(fake.makeBucketBuckets); got != 2 {
		t.Fatalf("expected 2 MakeBucket calls, got %d", got)
	}
}

func TestEnsureBucketWithRetry_RetriesWithoutSleeping(t *testing.T) {
	calls := 0
	fake := &fakeMinioClient{
		BucketExistsFn: func(_ context.Context, _ string) (bool, error) {
			calls++
			if calls < 3 {
				return false, errors.New("temporary failure")
			}

			return true, nil
		},
	}

	client := &MinIOClient{client: fake, config: &MinIOConfig{}}
	if err := client.ensureBucketWithRetry(context.Background(), "scanner-staging", 3, 0); err != nil {
		t.Fatalf("ensureBucketWithRetry: %v", err)
	}

	if calls != 3 {
		t.Fatalf("expected 3 BucketExists attempts, got %d", calls)
	}
}

func TestEnsureBucketWithRetry_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fake := &fakeMinioClient{
		BucketExistsFn: func(_ context.Context, _ string) (bool, error) {
			t.Fatalf("unexpected BucketExists call")
			return false, nil
		},
	}

	client := &MinIOClient{client: fake, config: &MinIOConfig{}}
	if err := client.ensureBucketWithRetry(ctx, "scanner-staging", 3, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestUploadFile_UsesStreamingWhenSizeUnknown(t *testing.T) {
	fake := &fakeMinioClient{}

	client := &MinIOClient{client: fake, config: &MinIOConfig{}}
	if err := client.UploadFile(context.Background(), "bucket", "path", bytes.NewReader([]byte("x")), -1); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	if got := fake.putObjectSizes; len(got) != 1 || got[0] != -1 {
		t.Fatalf("expected size -1, got %v", got)
	}
}

func TestUploadFile_PassesProvidedSize(t *testing.T) {
	fake := &fakeMinioClient{}

	client := &MinIOClient{client: fake, config: &MinIOConfig{}}
	if err := client.UploadFile(context.Background(), "bucket", "path", bytes.NewReader([]byte("x")), 123); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	if got := fake.putObjectSizes; len(got) != 1 || got[0] != 123 {
		t.Fatalf("expected size 123, got %v", got)
	}
}

func TestGetPresignedURL_UsesPublicClientWhenConfigured(t *testing.T) {
	internal := &fakeMinioClient{
		PresignFn: func(_ context.Context, _, _ string, _ time.Duration, _ url.Values) (*url.URL, error) {
			t.Fatalf("unexpected internal presign call")

			return nil, errors.New("unexpected internal presign call")
		},
	}

	publicURL, _ := url.Parse("https://public.example.com/presigned")
	public := &fakeMinioClient{
		PresignFn: func(_ context.Context, _, _ string, _ time.Duration, _ url.Values) (*url.URL, error) {
			return publicURL, nil
		},
	}

	client := &MinIOClient{
		client:       internal,
		publicClient: public,
		config:       &MinIOConfig{UseProxyURLs: false},
	}

	got, err := client.GetPresignedURL(context.Background(), "bucket", "path", time.Minute)
	if err != nil {
		t.Fatalf("GetPresignedURL: %v", err)
	}

	if got != publicURL.String() {
		t.Fatalf("expected %q, got %q", publicURL.String(), got)
	}

	if public.presignCalls != 1 {
		t.Fatalf("expected 1 public presign call, got %d", public.presignCalls)
	}
}

func TestGetPresignedURL_FallsBackToInternalClient(t *testing.T) {
	internalURL, _ := url.Parse("https://internal.example.com/presigned")
	internal := &fakeMinioClient{
		PresignFn: func(_ context.Context, _, _ string, _ time.Duration, _ url.Values) (*url.URL, error) {
			return internalURL, nil
		},
	}

	client := &MinIOClient{
		client: internal,
		config: &MinIOConfig{UseProxyURLs: false},
	}

	got, err := client.GetPresignedURL(context.Background(), "bucket", "path", time.Minute)
	if err != nil {
		t.Fatalf("GetPresignedURL: %v", err)
	}

	if got != internalURL.String() {
		t.Fatalf("expected %q, got %q", internalURL.String(), got)
	}
}

func TestDownloadFile_ClosesObjectWhenStatFails(t *testing.T) {
	obj := newFakeObject([]byte("hi"), errors.New("no such key"))

	fake := &fakeMinioClient{
		GetObjectFn: func(_ context.Context, _, _ string, _ minio.GetObjectOptions) (minioObject, error) {
			return obj, nil
		},
	}

	client := &MinIOClient{client: fake, config: &MinIOConfig{}}
	if _, err := client.DownloadFile(context.Background(), "bucket", "path"); err == nil {
		t.Fatalf("expected error")
	}

	if !obj.closed {
		t.Fatalf("expected object to be closed on Stat failure")
	}
}

func TestDeleteFile_WrapsErrors(t *testing.T) {
	fake := &fakeMinioClient{
		RemoveObjectFn: func(_ context.Context, _, _ string, _ minio.RemoveObjectOptions) error {
			return errors.New("boom")
		},
	}

	client := &MinIOClient{client: fake, config: &MinIOConfig{}}
	if err := client.DeleteFile(context.Background(), "bucket", "path"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFileExists_HandlesNoSuchKey(t *testing.T) {
	fake := &fakeMinioClient{
		StatObjectFn: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			return minio.ObjectInfo{}, minio.ErrorResponse{Code: "NoSuchKey"}
		},
	}

	client := &MinIOClient{client: fake, config: &MinIOConfig{}}

	exists, err := client.FileExists(context.Background(), "bucket", "path")
	if err != nil {
		t.Fatalf("FileExists: %v", err)
	}

	if exists {
		t.Fatalf("expected exists=false for NoSuchKey")
	}
}
