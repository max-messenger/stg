package app

import (
	"go.uber.org/fx"

	"stg/docs"
	"stg/internal/app/clients"
	"stg/internal/app/repository"
	"stg/internal/app/router"
	"stg/internal/app/router/authctrl"
	"stg/internal/app/router/filectrl"
	"stg/internal/app/router/todolistctrl"
	"stg/internal/app/router/webhookctrl"
	"stg/internal/app/services"
	"stg/internal/app/subscriber"
	"stg/pkg/bgtasker"
	"stg/pkg/connections/kafka"
	"stg/pkg/connections/minio"
	"stg/pkg/connections/postgres"
	"stg/pkg/connections/rediscli"
	"stg/pkg/migrate"
)

// Modules application modules.
var Modules = fx.Options(
	// controllers
	webhookctrl.Module,
	todolistctrl.Module,
	filectrl.Module,
	authctrl.Module,

	// clients
	clients.Module,

	// repositories
	repository.Module,

	// connections
	postgres.Module,
	rediscli.Module,
	kafka.Module,
	minio.Module,

	// bgtasker
	bgtasker.Module,

	// migrations
	migrate.Module,

	// services
	services.Module,

	subscriber.Module,

	router.Module,
	docs.Module,
)
