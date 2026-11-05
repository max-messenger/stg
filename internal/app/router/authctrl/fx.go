package authctrl

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/router"
	"stg/internal/app/services/auth"
)

const name = "authctrl"

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

	AuthService *auth.Service
	AuthConfig  auth.Config
}

type AdapterOut struct {
	fx.Out

	AuthService AuthService
}

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		AuthService: in.AuthService,
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
