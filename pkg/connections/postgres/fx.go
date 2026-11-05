//nolint:ireturn

package postgres

import (
	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module(
	"postgres_pool",
	fx.Provide(
		NewPostgresPool,
		configProvider("postgres"),
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("postgres_pool")
	}),
	fx.Invoke(
		func(lc fx.Lifecycle, pp *Pool) {
			lc.Append(fx.Hook{
				OnStop: pp.Stop,
			})
		},
	),
)

func configProvider(name string) func(cp config.Provider) (PoolConfig, error) {
	return func(cp config.Provider) (PoolConfig, error) {
		cfg := PoolConfig{}

		if err := cp.Get(name).Populate(&cfg); err != nil {
			return PoolConfig{}, err
		}

		return cfg, nil
	}

}

type PoolParams struct {
	fx.In

	Cfgs PoolConfig

	Logger *zap.Logger

	BeforeConnectCallbacks []BeforeConnectCallback `group:"pg_callbacks"`
	AfterConnectCallbacks  []AfterConnectCallback  `group:"pg_callbacks"`
}
