package filestore

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"

	"stg/internal/app/domain"
)

//go:generate go tool mockgen -source=filestore.go -destination=filestore_mocks_test.go -package=filestore_test

type Repository interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (domain.FileDownload, error)
	Remove(ctx context.Context, key string) error
}

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service {
	return &Service{
		repo: r,
	}
}

func (s *Service) Upload(
	ctx context.Context,
	filename string,
	reader io.Reader,
	size int64,
	contentType string,
) (domain.FileUpload, error) {
	key := generateKey(filename)

	err := s.repo.Put(ctx, key, reader, size, contentType)
	if err != nil {
		return domain.FileUpload{}, fmt.Errorf("upload file: %w", err)
	}

	return domain.FileUpload{Key: key}, nil
}

func (s *Service) Download(ctx context.Context, key string) (domain.FileDownload, error) {
	result, err := s.repo.Get(ctx, key)
	if err != nil {
		return domain.FileDownload{}, fmt.Errorf("download file: %w", err)
	}

	return result, nil
}

func (s *Service) Delete(ctx context.Context, key string) error {
	err := s.repo.Remove(ctx, key)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

func generateKey(filename string) string {
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]

	if base == "" {
		base = "file"
	}

	return fmt.Sprintf("%s-%s%s", base, uuid.NewString(), ext)
}
