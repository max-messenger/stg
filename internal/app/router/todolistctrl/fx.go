package todolistctrl

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/router"
	"stg/internal/app/services/todolist"
)

const name = "todolistctrl"

var Module = fx.Module(
	name,
	fx.Provide(
		NewController,
		Adapter,
		AdapterControllerOut,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named(name)
	}),
)

type AdapterIn struct {
	fx.In

	TodoService *todolist.Service
}

type AdapterOut struct {
	fx.Out

	TodoService TodoService
}

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		TodoService: in.TodoService,
	}
}

type ControllerOut struct {
	fx.Out

	Controller router.Controller `group:"controller"`
}

func AdapterControllerOut(ctrl *Controller) ControllerOut {
	return ControllerOut{
		Controller: ctrl,
	}
}
