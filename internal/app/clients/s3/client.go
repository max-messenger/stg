package s3

//go:generate go tool mockgen -source=client.go -destination=store_mock_test.go -package=s3_test

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"stg/internal/app/domain"
	"stg/pkg/telemetry"
)

type S3 interface {
	PutObject(context.Context, string, string, io.Reader, int64, minio.PutObjectOptions) (minio.UploadInfo, error)
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) (err error)
}

type Client struct {
	logger *zap.Logger
	client S3
	bucket string
}

func NewClient(log *zap.Logger, c S3, bucket string) *Client {
	return &Client{
		logger: log,
		client: c,
		bucket: bucket,
	}
}

func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	var err error

	defer func(t time.Time) {
		metricRequestsTotal.WithLabelValues("upload", telemetry.ErrLabelValue(err)).Inc()
		metricRequestsDuration.WithLabelValues("upload").Observe(time.Since(t).Seconds())
	}(time.Now())

	_, err = c.client.PutObject(ctx, c.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}

func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, domain.ObjectMeta, error) {
	var err error

	defer func(t time.Time) {
		metricRequestsTotal.WithLabelValues("download", telemetry.ErrLabelValue(err)).Inc()
		metricRequestsDuration.WithLabelValues("download").Observe(time.Since(t).Seconds())
	}(time.Now())

	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, domain.ObjectMeta{}, fmt.Errorf("get object: %w", err)
	}

	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()

		return nil, domain.ObjectMeta{}, fmt.Errorf("stat object: %w", err)
	}

	return obj, domain.ObjectMeta{
		ContentType: info.ContentType,
		Size:        info.Size,
	}, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	var err error

	defer func(t time.Time) {
		metricRequestsTotal.WithLabelValues("delete", telemetry.ErrLabelValue(err)).Inc()
		metricRequestsDuration.WithLabelValues("delete").Observe(time.Since(t).Seconds())
	}(time.Now())

	err = c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("remove object: %w", err)
	}

	return nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}

	if !exists {
		err = c.client.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}

	return nil
}
