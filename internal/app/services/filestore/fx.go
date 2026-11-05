package filestore

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/repository/filerepo"
)

const name = "filestore"

var Module = fx.Module(
	name,
	fx.Provide(
		NewService,
		Adapter,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named(name)
	}),
)

type AdapterIn struct {
	fx.In

	FileRepository *filerepo.Repository
}

type AdapterOut struct {
	fx.Out

	FileRepo Repository
}

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		FileRepo: in.FileRepository,
	}
}
