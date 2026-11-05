package clients

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/clients/maxbot"
	"stg/internal/app/clients/s3"
)

var Module = fx.Module(
	"clients",
	fx.Options(
		maxbot.Module,
		s3.Module,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("clients")
	}),
)
