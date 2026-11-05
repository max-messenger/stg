package subscriber

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/clients/maxbot"
	"stg/internal/app/services/bot"
	"stg/pkg/grace"
)

var Module = fx.Module(
	"subscriber",
	fx.Provide(
		New,
		Adapter,
		graceAdapter,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("subscriber")
	}),
)

type AdapterIn struct {
	fx.In

	Messages *maxbot.Client
	Bot      *bot.MaxBot
}

type AdapterOut struct {
	fx.Out

	Messages Messages
	Bot      Bot
}

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		Messages: in.Messages,
		Bot:      in.Bot,
	}
}

type graceOut struct {
	fx.Out

	Service grace.Service `group:"grace"`
}

func graceAdapter(sub *Subscriber) graceOut {
	return graceOut{
		Service: sub,
	}
}
