package webhookctrl

import (
	"go.uber.org/fx"

	"stg/internal/app/clients/maxbot"
	"stg/pkg/server"
)

const name = "webhookctrl"

var Module = fx.Module(
	name,
	fx.Provide(
		NewController,
		Adapter,
		AdapterControllerOut,
	),
)

type AdapterIn struct {
	fx.In

	Handler *maxbot.Client
}

type AdapterOut struct {
	fx.Out

	Handler Handler
}

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		Handler: in.Handler,
	}
}

type ControllerOut struct {
	fx.Out

	Controller server.Controller `group:"controller"`
}

func AdapterControllerOut(ctrl *Controller) ControllerOut {
	return ControllerOut{
		Controller: ctrl,
	}
}
