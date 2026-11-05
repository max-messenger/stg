package filerepo

import (
	"go.uber.org/fx"

	"stg/internal/app/clients/s3"
)

const name = "filerepo"

var Module = fx.Module(
	name,
	fx.Provide(
		NewRepository,
		Adapter,
	),
)

type (
	AdapterIn struct {
		fx.In

		Store *s3.Client
	}

	AdapterOut struct {
		fx.Out

		Store Store
	}
)

func Adapter(in AdapterIn) (AdapterOut, error) {
	return AdapterOut{
		Store: in.Store,
	}, nil
}
