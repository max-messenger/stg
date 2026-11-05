package services

import (
	"go.uber.org/fx"

	"stg/internal/app/services/auth"
	"stg/internal/app/services/bot"
	"stg/internal/app/services/filestore"
	"stg/internal/app/services/todolist"
	"stg/pkg/locker"
)

var Module = fx.Module(
	"services",
	fx.Options(
		bot.Module,
		filestore.Module,
		todolist.Module,
		locker.Module,
		auth.Module,
	),
)
