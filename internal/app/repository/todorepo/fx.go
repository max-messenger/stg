package todorepo

import (
	"github.com/huandu/go-sqlbuilder"
	"go.uber.org/fx"
)

func init() {
	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL
}

var Module = fx.Module(
	"todorepo",
	fx.Provide(
		NewRepository,
	),
)
