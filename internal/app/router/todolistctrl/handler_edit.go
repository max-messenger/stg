package todolistctrl

import (
	"net/http"

	"stg/internal/app/domain"
	"stg/internal/app/router"
	"stg/pkg/marshaler"
)

type EditRequest struct {
	ID      int64  `json:"id"`
	UserID  int64  `json:"user_id"`
	Message string `json:"message"`
}

func (r EditRequest) ToDomain() domain.Todo {
	return domain.Todo{
		ID:      r.ID,
		UserID:  r.UserID,
		Message: r.Message,
	}
}

type EditResponse struct {
	ID int64 `json:"id"`
}

// editHandler godoc
//
// @Summary		edit todo
// @Description	update todo message
// @Tags		todo
// @Accept		json
// @Produce		json
// @Param       Authorization  header    string  true  "Bearer token"
// @Param       request body     EditRequest false "request body"
// @Success		200		{object} EditResponse
// @Failure		400		{object} string
// @Failure		401		{object} string
// @Failure		403		{object} string
// @Failure		500		{object} string
// @Router		/v1/todos/edit [put]
// .
func (c *Controller) editHandler(w http.ResponseWriter, r *http.Request) {
	var request EditRequest

	err := marshaler.LoadJSONFromReader(r.Body, &request)
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	id, err := c.srv.Edit(r.Context(), request.ToDomain())
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	router.WriteSuccess(w, EditResponse{ID: id})
}
