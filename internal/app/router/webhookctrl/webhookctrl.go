package webhookctrl

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler interface {
	UpdateHandler() (string, http.HandlerFunc)
}

type Controller struct {
	handler Handler
}

func NewController(h Handler) *Controller {
	return &Controller{
		handler: h,
	}
}

func (c *Controller) Register(r chi.Router) {
	r.Handle(c.handler.UpdateHandler())
}
