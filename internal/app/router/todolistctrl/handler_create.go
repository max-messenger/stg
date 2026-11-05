package todolistctrl

import (
	"net/http"

	"stg/internal/app/domain"
	"stg/internal/app/router"
	"stg/pkg/marshaler"
)

type CreateRequest struct {
	UserID  int64  `json:"user_id"`
	Message string `json:"message"`
}

func (r CreateRequest) ToDomain() domain.Todo {
	return domain.Todo{
		UserID:  r.UserID,
		Message: r.Message,
	}
}

type CreateResponse struct {
	ID int64 `json:"id"`
}

// createHandler godoc
//
// @Summary		add todo
// @Description	create new todo record
// @Tags		todo
// @Accept		json
// @Produce		json
// @Param       Authorization  header    string  true  "Bearer token"
// @Param       request body     CreateRequest false "request body"
// @Success		200		{object} CreateResponse
// @Failure		400		{object} string
// @Failure		401		{object} string
// @Failure		403		{object} string
// @Failure		500		{object} string
// @Router		/v1/todos/create [post]
// .
func (c *Controller) createHandler(w http.ResponseWriter, r *http.Request) {
	var request CreateRequest

	err := marshaler.LoadJSONFromReader(r.Body, &request)
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	id, err := c.srv.Create(r.Context(), request.ToDomain())
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	router.WriteSuccess(w, CreateResponse{ID: id})
}
