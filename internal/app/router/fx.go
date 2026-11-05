package router

import (
	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/pkg/http/middlewares"
)

const moduleName = "httpsrv"

var Module = fx.Module(
	moduleName,
	fx.Provide(
		NewRouter,
		ControllersAdapter,
		NewAuthMiddleware,
		corsConfigProvider("cors"),
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named(moduleName)
	}),
)

func corsConfigProvider(cfgName string) func(cp config.Provider) (middlewares.CORSConfig, error) {
	return func(cp config.Provider) (middlewares.CORSConfig, error) {
		cfg := middlewares.CORSConfig{}

		if err := cp.Get(cfgName).Populate(&cfg); err != nil {
			return middlewares.CORSConfig{}, err
		}

		return cfg, nil
	}
}

type (
	ControllersIn struct {
		fx.In

		Controllers []Controller `group:"controller"`
	}

	ControllersOut struct {
		fx.Out

		Controllers Controllers
	}
)

func ControllersAdapter(in ControllersIn) ControllersOut {
	return ControllersOut{
		Controllers: in.Controllers,
	}
}
