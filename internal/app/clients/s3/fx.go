package s3

import (
	"github.com/minio/minio-go/v7"
	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const name = "s3"

var Module = fx.Module(
	name,
	fx.Provide(
		func(log *zap.Logger, c *minio.Client, cfg Config) *Client {
			return NewClient(log, c, cfg.Bucket)
		},
		configProvider(name),
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named(name)
	}),
	fx.Invoke(
		func(lc fx.Lifecycle, c *Client) {
			lc.Append(fx.Hook{
				OnStart: c.EnsureBucket,
			})
		},
	),
)

func configProvider(cfgName string) func(cp config.Provider) (Config, error) {
	return func(cp config.Provider) (Config, error) {
		cfg := Config{}

		if err := cp.Get(cfgName).Populate(&cfg); err != nil {
			return Config{}, err
		}

		return cfg, nil
	}
}
