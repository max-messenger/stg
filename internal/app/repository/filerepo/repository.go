package filerepo

//go:generate go tool mockgen -source=repository.go -destination=store_mock_test.go -package=filerepo_test

import (
	"context"
	"fmt"
	"io"

	"stg/internal/app/domain"
)

type Store interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, domain.ObjectMeta, error)
	Delete(ctx context.Context, key string) error
}

type Repository struct {
	client Store
}

func NewRepository(client Store) *Repository {
	return &Repository{
		client: client,
	}
}

func (r *Repository) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	err := r.client.Upload(ctx, key, reader, size, contentType)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, key string) (domain.FileDownload, error) {
	reader, meta, err := r.client.Download(ctx, key)
	if err != nil {
		return domain.FileDownload{}, fmt.Errorf("download file: %w", err)
	}

	return domain.FileDownload{
		Reader:      reader,
		ContentType: meta.ContentType,
		Size:        meta.Size,
	}, nil
}

func (r *Repository) Remove(ctx context.Context, key string) error {
	err := r.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}
