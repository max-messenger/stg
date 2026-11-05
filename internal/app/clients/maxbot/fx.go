package maxbot

import (
	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/pkg/grace"
	"stg/pkg/http/client"
)

var Module = fx.Module(
	"maxbot",
	fx.Provide(
		New,
		adapter,
		graceAdapter,
		configProvider("max_bot"),
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("maxbot")
	}),
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

func adapter(log *zap.Logger) HTTPClient {
	opts := []client.Option{
		client.WithLogger(log.Named("maxbot")),
	}

	return client.New("maxbot", opts...)
}

type graceOut struct {
	fx.Out

	Service grace.Service `group:"grace"`
}

func graceAdapter(sub *Client) graceOut {
	return graceOut{
		Service: sub,
	}
}
