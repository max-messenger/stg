package filectrl

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"stg/internal/app/router"
)

var errEmptyKey = errors.New("empty file key")

// downloadHandler godoc
//
// @Summary		download file
// @Description	download a file by its key
// @Tags		files
// @Produce		octet-stream
// @Param		key path string true "file key"
// @Success		200		{file} binary
// @Failure		400		{object} string
// @Failure		404		{object} string
// @Router		/v1/files/{key} [get]
// .
func (c *Controller) downloadHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		router.WriteError(w, http.StatusBadRequest, errEmptyKey)

		return
	}

	result, err := c.srv.Download(r.Context(), key)
	if err != nil {
		router.WriteError(w, http.StatusNotFound, err)

		return
	}
	defer result.Reader.Close()

	contentType := result.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+sanitizeFilename(key)+"\"")
	w.WriteHeader(http.StatusOK)

	buf, ok := copyBufferPool.Get().(*[]byte)
	if !ok {
		c.logger.Error("failed to get buffer from pool")

		return
	}
	defer copyBufferPool.Put(buf)

	if _, err := io.CopyBuffer(w, result.Reader, *buf); err != nil {
		c.logger.Error("copy file to response", zap.Error(err))
	}
}
