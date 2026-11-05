package auth

import (
	"go.uber.org/config"
	"go.uber.org/fx"

	"stg/internal/app/repository/tokenrepo"
)

const serviceName = "auth"

var Module = fx.Module(
	serviceName,
	fx.Provide(
		New,
		AdapterTokenRepo,
		configProvider(serviceName),
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

type TokenRepoIn struct {
	fx.In

	TokenRepo *tokenrepo.Repository
}

type TokenRepoOut struct {
	fx.Out

	Repo TokenRepo
}

func AdapterTokenRepo(in TokenRepoIn) TokenRepoOut {
	return TokenRepoOut{
		Repo: in.TokenRepo,
	}
}
