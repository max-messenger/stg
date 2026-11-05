package filectrl

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"stg/internal/app/router"
)

// deleteHandler godoc
//
// @Summary		delete file
// @Description	delete a file by its key
// @Tags		files
// @Produce		json
// @Param		key path string true "file key"
// @Success		200		{object} string
// @Failure		400		{object} string
// @Failure		404		{object} string
// @Router		/v1/files/{key} [delete]
// .
func (c *Controller) deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		router.WriteError(w, http.StatusBadRequest, errEmptyKey)

		return
	}

	err := c.srv.Delete(r.Context(), key)
	if err != nil {
		router.WriteError(w, http.StatusNotFound, err)

		return
	}

	router.WriteSuccess(w, nil)
}
