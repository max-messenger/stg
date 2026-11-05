package tokenrepo

import (
	"go.uber.org/config"
	"go.uber.org/fx"

	"stg/pkg/connections/rediscli"
)

const moduleName = "tokenrepo"

var Module = fx.Module(
	moduleName,
	fx.Provide(
		NewRepository,
		Adapter,
		configProvider("token_repo"),
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

type (
	AdapterIn struct {
		fx.In

		Store *rediscli.Pool
	}

	AdapterOut struct {
		fx.Out

		Store Store
	}
)

func Adapter(in AdapterIn) (AdapterOut, error) {
	client, err := in.Store.GetPool("main")
	if err != nil {
		return AdapterOut{}, err
	}

	return AdapterOut{
		Store: client,
	}, nil
}
