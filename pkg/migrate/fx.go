package migrate

import (
	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module(
	"migrate",
	fx.Provide(
		New,
		configProvider("migrate"),
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("migrate")
	}),
	fx.Invoke(
		func(lc fx.Lifecycle, mg *Migrations) {
			lc.Append(fx.Hook{
				OnStart: mg.Start,
			})
		},
	),
)

func configProvider(name string) func(cp config.Provider) (Config, error) {
	return func(cp config.Provider) (Config, error) {
		cfg := Config{}

		if err := cp.Get(name).Populate(&cfg); err != nil {
			return Config{}, err
		}

		return cfg, nil
	}
}
