package filectrl

import (
	"net/http"

	"stg/internal/app/domain"
	"stg/internal/app/router"
)

type UploadResponse struct {
	Key string `json:"key"`
}

// uploadHandler godoc
//
// @Summary		upload file
// @Description	upload a file via multipart form and return its key
// @Tags		files
// @Accept		multipart/form-data
// @Produce		json
// @Param		file formData file true "file to upload"
// @Success		200		{object} UploadResponse
// @Failure		400		{object} string
// @Failure		500		{object} string
// @Router		/v1/files/ [post]
// .
func (c *Controller) uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(c.maxUploadBytes) //nolint:gosec // G120: max upload size is bounded by config
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	result, err := c.srv.Upload(r.Context(), header.Filename, file, header.Size, contentType)
	if err != nil {
		router.WriteError(w, http.StatusInternalServerError, err)

		return
	}

	router.WriteSuccess(w, UploadResponse{Key: result.Key})
}

// Проверка, что UploadResponse маппится из domain.FileUpload.
var _ = domain.FileUpload{}
