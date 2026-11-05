package filectrl

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/router"
	"stg/internal/app/services/filestore"
)

const name = "filectrl"

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

	FileService *filestore.Service
}

type AdapterOut struct {
	fx.Out

	Srv FileService
}

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		Srv: in.FileService,
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
