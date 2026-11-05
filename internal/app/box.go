package app

import (
	"context"

	_ "go.uber.org/automaxprocs" // add automaxprocs
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"stg/pkg/config"
	"stg/pkg/grace"
	"stg/pkg/health"
	"stg/pkg/http/client"
	"stg/pkg/info"
	"stg/pkg/logger"
	"stg/pkg/server"
	"stg/pkg/telemetry"
)

var defaultModules = fx.Options(
	grace.Module,
	health.Module,
	server.Module,
	info.Module,
	client.Module,
	telemetry.Module,
)

type Box struct {
	appName   string
	cfgFile   string
	fxMoudles fx.Option
}

func NewBox(modules fx.Option, opts ...BoxOption) *Box {
	b := &Box{
		appName: "undefined",
		cfgFile: "config.yaml",
	}

	for _, opt := range opts {
		opt(b)
	}

	b.fxMoudles = fx.Options(
		defaultModules,
		modules,
	)

	return b
}

func (b *Box) CreateApp() *fx.App {
	return fx.New(
		fx.Provide(
			logger.NewLogger(b.appName),
			config.NewConfig(b.appName, b.cfgFile), // provide config
		),
		// zap logger
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		// invoke base background services
		fx.Invoke(func(lc fx.Lifecycle, gr *grace.ServicePool) {
			lc.Append(fx.Hook{
				OnStart: gr.Start,
				OnStop:  gr.Stop,
			})
		}),
		b.fxMoudles, // can contain invokes with depenency on default/graceful services
		fx.Invoke(func(lc fx.Lifecycle, h *health.Health) {
			lc.Append(fx.Hook{
				OnStart: func(_ context.Context) error {
					h.SetReady(true) // set application ready

					return nil
				},
				OnStop: func(_ context.Context) error {
					h.SetReady(false) // set application stopped

					return nil
				},
			})
		}),
	)
}
