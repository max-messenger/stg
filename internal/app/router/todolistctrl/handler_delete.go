package todolistctrl

import (
	"net/http"

	"stg/internal/app/domain"
	"stg/internal/app/router"
	"stg/pkg/marshaler"
)

type DeleteRequest struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`
}

func (r DeleteRequest) ToDomain() domain.Todo {
	return domain.Todo{
		ID:     r.ID,
		UserID: r.UserID,
	}
}

// deleteHandler godoc
//
// @Summary		delete todo
// @Description	delete by id
// @Tags		todo
// @Accept		json
// @Produce		json
// @Param       Authorization  header    string  true  "Bearer token"
// @Param       request body     DeleteRequest false "request body"
// @Success		200		{object} int64
// @Failure		400		{object} string
// @Failure		401		{object} string
// @Failure		403		{object} string
// @Failure		500		{object} string
// @Router		/v1/todos/delete [post]
// .
func (c *Controller) deleteHandler(w http.ResponseWriter, r *http.Request) {
	var request DeleteRequest

	err := marshaler.LoadJSONFromReader(r.Body, &request)
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	err = c.srv.Delete(r.Context(), request.ToDomain())
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	router.WriteSuccess(w, request.ID)
}
