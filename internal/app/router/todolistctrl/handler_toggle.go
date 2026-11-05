package todolistctrl

import (
	"net/http"

	"stg/internal/app/domain"
	"stg/internal/app/router"
	"stg/pkg/marshaler"
)

type ToggleRequest struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`
	Done   bool  `json:"done"`
}

func (r ToggleRequest) ToDomain() domain.Todo {
	return domain.Todo{
		ID:     r.ID,
		UserID: r.UserID,
		Done:   r.Done,
	}
}

// toggleHandler godoc
//
// @Summary		toggle todo
// @Description	toggle todo done status
// @Tags		todo
// @Accept		json
// @Produce		json
// @Param       Authorization  header    string  true  "Bearer token"
// @Param       request body     ToggleRequest false "request body"
// @Success		200		{object} int64
// @Failure		400		{object} string
// @Failure		401		{object} string
// @Failure		403		{object} string
// @Failure		500		{object} string
// @Router		/v1/todos/toggle [post]
// .
func (c *Controller) toggleHandler(w http.ResponseWriter, r *http.Request) {
	var request ToggleRequest

	err := marshaler.LoadJSONFromReader(r.Body, &request)
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	err = c.srv.Toggle(r.Context(), request.ToDomain())
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	router.WriteSuccess(w, request.ID)
}
