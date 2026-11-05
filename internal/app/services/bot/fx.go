package bot

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/clients/maxbot"
	"stg/internal/app/repository/staterepo"
	"stg/internal/app/repository/todorepo"
	"stg/pkg/locker"
)

var Module = fx.Module(
	"bot",
	fx.Provide(
		New,
		Adapter,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("bot")
	}),
)

type (
	AdapterIn struct {
		fx.In

		Client *maxbot.Client
		Store  *staterepo.Repository
		Repo   *todorepo.Repository
		Locker *locker.Locker
	}

	AdapterOut struct {
		fx.Out

		Client Client
		Store  Store
		Repo   Repository
	}
)

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		Client: in.Client,
		Store:  in.Store,
		Repo:   in.Repo,
	}
}
