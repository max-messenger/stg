package todolistctrl

import (
	"net/http"
	"time"

	"stg/internal/app/domain"
	"stg/internal/app/router"
	"stg/pkg/marshaler"
)

type ListRequest struct {
	UserID int64 `json:"user_id"`
}

type ListResponse struct {
	Todos []domain.Todo `json:"todos"`
}

type TodoItem struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
	Done    bool   `json:"done"`
	Created string `json:"created"`
}

func (l ListResponse) ToItems() []TodoItem {
	result := make([]TodoItem, 0, len(l.Todos))
	for _, v := range l.Todos {
		result = append(result, TodoItem{
			ID:      v.ID,
			Message: v.Message,
			Done:    v.Done,
			Created: v.Created.Format(time.RFC3339),
		})
	}

	return result
}

// listHandler godoc
//
// @Summary		get todo list
// @Description	get all todo records
// @Tags		todo
// @Accept		json
// @Produce		json
// @Param       Authorization  header    string  true  "Bearer token"
// @Success		200		{object} ListResponse
// @Failure		400		{object} string
// @Failure		401		{object} string
// @Failure		403		{object} string
// @Failure		500		{object} string
// @Router		/v1/todos/list [post]
// .
func (c *Controller) listHandler(w http.ResponseWriter, r *http.Request) {
	var req ListRequest

	err := marshaler.LoadJSONFromReader(r.Body, &req)
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	records, err := c.srv.List(r.Context(), req.UserID)
	if err != nil {
		router.WriteError(w, http.StatusBadRequest, err)

		return
	}

	res := ListResponse{
		Todos: records,
	}

	router.WriteSuccess(w, res.ToItems())
}
