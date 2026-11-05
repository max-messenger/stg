package filectrl

import (
	"context"
	"io"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"stg/internal/app/domain"
	"stg/internal/app/services/filestore"
)

type FileService interface {
	Upload(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) (domain.FileUpload, error)
	Download(ctx context.Context, key string) (domain.FileDownload, error)
	Delete(ctx context.Context, key string) error
}

type Controller struct {
	logger         *zap.Logger
	srv            FileService
	maxUploadBytes int64
}

const defaultMaxUploadBytes = 10 << 20 // 10MB

func NewController(l *zap.Logger, s FileService, opts ...Option) *Controller {
	c := &Controller{
		logger:         l,
		srv:            s,
		maxUploadBytes: defaultMaxUploadBytes,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type Option func(*Controller)

func (c *Controller) Register(r chi.Router) {
	r.Route("/v1/files", func(r chi.Router) {
		r.Post("/", c.uploadHandler)
		r.Get("/{key}", c.downloadHandler)
		r.Delete("/{key}", c.deleteHandler)
	})
}

// Проверка, что *filestore.Service удовлетворяет FileService.
var _ FileService = (*filestore.Service)(nil)
