package todolistctrl

import (
	"context"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"stg/internal/app/domain"
	"stg/internal/app/router"
)

type TodoService interface {
	Create(ctx context.Context, r domain.Todo) (int64, error)
	Edit(ctx context.Context, r domain.Todo) (int64, error)
	List(ctx context.Context, userID int64) ([]domain.Todo, error)
	Delete(ctx context.Context, r domain.Todo) error
	Toggle(ctx context.Context, r domain.Todo) error
}

type Controller struct {
	logger *zap.Logger
	srv    TodoService
	authMw router.AuthMiddleware
}

func NewController(l *zap.Logger, s TodoService, authMw router.AuthMiddleware) *Controller {
	return &Controller{
		logger: l,
		srv:    s,
		authMw: authMw,
	}
}

func (c *Controller) Register(r chi.Router) {
	r.Route("/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(c.authMw)
			r.Post("/todos/list", c.listHandler)
			r.Post("/todos/create", c.createHandler)
			r.Put("/todos/edit", c.editHandler)
			r.Post("/todos/delete", c.deleteHandler)
			r.Post("/todos/toggle", c.toggleHandler)
		})
	})
}
