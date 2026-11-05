package repository

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/repository/filerepo"
	"stg/internal/app/repository/staterepo"
	"stg/internal/app/repository/todorepo"
	"stg/internal/app/repository/tokenrepo"
)

var Module = fx.Module(
	"repository",
	fx.Options(
		filerepo.Module,
		staterepo.Module,
		todorepo.Module,
		tokenrepo.Module,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("repository")
	}),
)
