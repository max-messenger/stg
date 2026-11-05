package todolist

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/repository/todorepo"
	"stg/internal/app/services/auth"
)

const serviceName = "todolist"

var Module = fx.Module(
	serviceName,
	fx.Provide(
		NewService,
		Adapter,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named(serviceName)
	}),
)

type (
	AdapterIn struct {
		fx.In

		Repository *todorepo.Repository
		Auth       *auth.Service
	}

	AdapterOut struct {
		fx.Out

		Repository Repository
		Auth       Auth
	}
)

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{

		Repository: in.Repository,
		Auth:       in.Auth,
	}
}
