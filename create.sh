#!/bin/bash

# Проверка наличия аргумента
if [ -z "$1" ]; then
    echo "Использование: ./create.sh <new_project_name>"
    exit 1
fi

NEW_PROJECT_NAME="$1"
CURRENT_DIR="$(pwd)"
BASE_DIR="$(basename "$CURRENT_DIR")"
NEW_DIR="${CURRENT_DIR}/../${NEW_PROJECT_NAME}"

# Проверка, что мы в папке stg
if [ "$BASE_DIR" != "stg" ]; then
    echo "Ошибка: Скрипт должен быть запущен из папки stg"
    exit 1
fi

# Проверка существования целевой папки
if [ -d "$NEW_DIR" ]; then
    echo "Ошибка: Папка $NEW_DIR уже существует"
    exit 1
fi

echo "Создание копии проекта..."
echo "Исходная директория: $CURRENT_DIR"
echo "Новая директория: $NEW_DIR"
echo "Новое имя проекта: $NEW_PROJECT_NAME"
echo ""

# Копирование проекта
mkdir "$NEW_DIR" || exit 1
rsync -a --exclude='/.coverage/' --exclude='/.codegraph/' "$CURRENT_DIR/" "$NEW_DIR/" || exit 1

cd ${NEW_DIR}

# Сначала переименовываем папки
mv ./cmd/stg ./cmd/app
mv ./config/stg ./config/${NEW_PROJECT_NAME}

# Удаление лишних файлов и папок в internal/app
echo "Удаление лишних файлов в internal/app..."

# Удаляем доменные модели примера, сохраняя message.go для MAX-клиента и модели auth.
rm -f ./internal/app/domain/callback*
rm -f ./internal/app/domain/s3*
rm -f ./internal/app/domain/todolist.go
rm -f ./internal/app/domain/file.go
rm -f ./internal/app/domain/init_data.go
rm -f ./internal/app/config.go

# Удаляем папку repository
rm -rf ./internal/app/repository/filerepo
rm -rf ./internal/app/repository/staterepo
rm -rf ./internal/app/repository/todorepo
echo '' > ./internal/app/repository/fx.go

# Удаляем папку services
rm -rf ./internal/app/services/bot
rm -rf ./internal/app/services/filestore
rm -rf ./internal/app/services/todolist
echo '' > ./internal/app/services/fx.go

# Удаляем папку docs/http
rm -rf ./docs/http

# Удаляем папку todolistctrl из router
rm -rf ./internal/app/router/todolistctrl
rm -rf ./internal/app/router/filectrl

# Удаляем s3 клиента
rm -rf ./internal/app/clients/s3

# Удаляем тесты и testkit
rm -rf ./tests
rm -rf ./testkit

# Сохраняем auth guideline: auth-компоненты остаются в новом приложении.

# Очищаем папку db/migrations, оставляем только .gitkeep
rm -rf ./db/migrations/*
# Создаём .gitkeep если его нет
touch ./db/migrations/.gitkeep

# Перезаписываем fx.go упрощенной версией
echo "Перезапись services/fx.go..."
cat > ./internal/app/repository/fx.go << 'REPOEOF'
package repository

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/repository/tokenrepo"
)

var Module = fx.Module(
	"repository",
	fx.Options(
		tokenrepo.Module,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("repository")
	}),
)
REPOEOF

# Перезаписываем fx.go упрощенной версией
echo "Перезапись services/fx.go..."
cat > ./internal/app/services/fx.go << 'SERVICESEOF'
package services

import (
	"go.uber.org/fx"

	"stg/internal/app/services/auth"
)

var Module = fx.Module(
	"services",
	fx.Options(
		auth.Module,
	),
)
SERVICESEOF

# Subscriber в новом приложении только логирует обновления MAX.
echo "Перезапись subscriber/fx.go..."
cat > ./internal/app/subscriber/fx.go << 'SUBSCRIBERFXEOF'
package subscriber

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/clients/maxbot"
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
}

type AdapterOut struct {
	fx.Out

	Messages Messages
}

func Adapter(in AdapterIn) AdapterOut {
	return AdapterOut{
		Messages: in.Messages,
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
SUBSCRIBERFXEOF

echo "Перезапись subscriber/subscriber.go..."
cat > ./internal/app/subscriber/subscriber.go << 'SUBSCRIBEREOF'
package subscriber

import (
	"context"
	"sync"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type Messages interface {
	Updates() <-chan model.Update
}

type Subscriber struct {
	logger      *zap.Logger
	botMessages Messages
	quit        chan struct{}
	wg          sync.WaitGroup
}

func New(logger *zap.Logger, botMessages Messages) *Subscriber {
	return &Subscriber{
		logger:      logger,
		botMessages: botMessages,
		quit:        make(chan struct{}),
	}
}

func (s *Subscriber) Start(_ context.Context) error {
	s.logger.Info("update subscriber started")
	s.wg.Go(func() {
		s.messagesConsumer()
	})

	return nil
}

func (s *Subscriber) Stop(_ context.Context) error {
	s.logger.Info("update subscriber stopping")
	close(s.quit)
	s.wg.Wait()
	s.logger.Info("update subscriber stopped")

	return nil
}

func (s *Subscriber) messagesConsumer() {
	ch := s.botMessages.Updates()
	for {
		select {
		case <-s.quit:
			return
		case upd, ok := <-ch:
			if !ok {
				s.logger.Info("update channel closed")

				return
			}
			s.logger.Debug("dispatching update",
				zap.String("update_type", string(upd.UpdateType)),
				zap.Int64("user_id", upd.UserID),
				zap.Int64("chat_id", upd.ChatID),
			)
			s.wg.Go(func() {
				_, span := otel.Tracer("subscriber").Start(context.Background(), "bot-message")
				defer span.End()

				defer func(t time.Time) {
					metricRequestsTotal.WithLabelValues("bot-message").Inc()
					metricRequestsDuration.WithLabelValues("bot-message").Observe(time.Since(t).Seconds())
				}(time.Now())

				s.logger.Debug("new message", zap.Any("message", upd)) // сюда писать обработчик, лог можно удалить
			})
		}
	}
}
SUBSCRIBEREOF

# Перезаписываем fx.go упрощенной версией
echo "Перезапись fx.go..."
cat > ./internal/app/fx.go << 'FXEOF'
package app

import (

	"go.uber.org/fx"

	"stg/docs"
	"stg/internal/app/clients"
	"stg/internal/app/repository"
	"stg/internal/app/router"
	"stg/internal/app/router/authctrl"
	"stg/internal/app/router/webhookctrl"
	"stg/internal/app/services"
	"stg/internal/app/subscriber"
	"stg/pkg/connections/rediscli"
)

// Modules application modules.
var Modules = fx.Options(
	// controllers
	authctrl.Module,
	webhookctrl.Module,

	// clients
	clients.Module,

	router.Module,
	docs.Module,

	rediscli.Module,
	repository.Module,

	services.Module,
	subscriber.Module,
)
FXEOF

# Перезаписываем router.go без CORS
echo "Перезапись router/router.go..."
cat > ./internal/app/router/router.go << 'ROUTEREOF'
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"stg/pkg/http/middlewares"
)

// Controller api controller.
type Controller interface {
	Register(r chi.Router)
}

type Controllers []Controller

// 	@title          STG API
// 	@version        1.0
// 	@description    This is a stg API.

//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8080
//	@BasePath	/

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func NewRouter(logger *zap.Logger, controllers Controllers) (http.Handler, error) {
	router := chi.NewRouter()

	router.Route("/", func(r chi.Router) {
		r.Use(
			middlewares.Metrics(),
			middlewares.AccessLog(logger),
			middlewares.Trace(),
		)
		for _, ctrl := range controllers {
			ctrl.Register(r)
		}
	})

	return router, nil
}
ROUTEREOF

# Перезаписываем router/fx.go без CORS config
echo "Перезапись router/fx.go..."
cat > ./internal/app/router/fx.go << 'ROUTERFXEOF'
package router

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module(
	"httpsrv",
	fx.Provide(
		NewRouter,
		ControllersAdapter,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("http_server")
	}),
)

type (
	ControllersIn struct {
		fx.In

		Controllers []Controller `group:"controller"`
	}

	ControllersOut struct {
		fx.Out

		Controllers Controllers
	}
)

func ControllersAdapter(in ControllersIn) ControllersOut {
	return ControllersOut{
		Controllers: in.Controllers,
	}
}
ROUTERFXEOF

# Перезаписываем config.tmpl.yaml без auth/cors
echo "Перезапись config.tmpl.yaml..."
cat > ./config/${NEW_PROJECT_NAME}/config.tmpl.yaml << 'CFGEOF'
app:
  hostname: ${HOSTNAME:"stg"}

logger:
  level: info
  format: json

servers:
  http:
    addr: :8080
  system:
    addr: :23569

subscriber:
  analytic_events:
    enabled: true
    poll_records: 1
    topics:
      - ${ANALYTIC_TOPIC:"analytic"}

max_bot:
  token: ${MAX_BOT_TOKEN:"token"}
  url: ${MAXBOT_URL:"http://localhost:8080"}
  path: ${MAXBOT_PATH:"/webhook"}
  subscription_type: ${MAXBOT_SUBSCRIPTION_TYPE:"poll"}

analytic:
  enabled: ${ANALYTIC_ENABLED:"false"}
  topic: ${ANALYTIC_TOPIC:"analytic"}
  extra_fields:
    app: stg

redis:
  main:
    key_prefix: "stg"
    addrs: ['${REDIS_ADDRS:"localhost:6379"}']

locker:
  pool_name: main

postgres:
  stg-sel:
    database_url: ${POSTGRES_CONNECTION_STRING_SEL:"postgres://postgres:postgres@localhost:5432/stg?default_query_exec_mode=exec"}
  stg-upd:
    database_url: ${POSTGRES_CONNECTION_STRING_UPD:"postgres://postgres:postgres@localhost:5432/stg?default_query_exec_mode=exec"}

migrate:
  pool_name: stg-upd

kafka:
  analytic:
    seeds:
      - localhost:19092
    allow_auto_topic_creation: true
    producer:
      disable_idempotent_write: true
      enable_compress: true

telemetry:
  collector:
    enabled: true
    endpoint: localhost:4317
    endpoint_type: grpc
    sampling_ratio: 1

CFGEOF

# Перезаписываем fx.go в clients без s3
echo "Перезапись clients/fx.go..."
cat > ./internal/app/clients/fx.go << 'CLIENTSFXEOF'
package clients

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/internal/app/clients/maxbot"
)

var Module = fx.Module(
	"clients",
	fx.Options(
		maxbot.Module,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("clients")
	}),
)
CLIENTSFXEOF

# Удаляем секцию "Пример приложения" из README.md
echo "Очистка README.md..."
sed -i '' '/### Пример приложения/,/^```$/d' ./README.md



# Замена "stg" на новое имя во всех текстовых файлах
# Используем sed с grep для фильтрации - только файлы, которые содержат "stg"
echo "Замена 'stg' на '${NEW_PROJECT_NAME}' в файлах..."
find . -type f \( -name "*.go" -o -name "*.yaml" -o -name "*.yml" -o -name "*.json" -o -name "*.md" -o -name "*.sh" -o -name "*.mod" -o -name "*.txt" -o -name "*.sql" -o -name ".gitignore" \) -not -path "./.git/*" -not -path "./vendor/*" | while read -r file; do
    if grep -q "stg" "$file" 2>/dev/null; then
        # Заменяем "stg" если перед ним НЕ буква/цифра и после него НЕ буква/цифра
        # Это позволит заменить "stg_redis", "my-stg-project", "postgres" останется как есть
        case "$file" in
            ./AGENTS.md|./.agents/*|./guidelines/*.md)
                # Имена скиллов и их каталогов остаются stg-*, пути приложения переименовываются.
                sed -i '' -E \
                    -e 's/stg-(add-http-endpoint|add-repository|implement-feature|invariants-review|\*)/__STG_SKILL_NAME__-\1/g' \
                    -e "s/(^|[^a-zA-Z0-9])stg([^a-zA-Z0-9]|$)/\1${NEW_PROJECT_NAME}\2/g" \
                    -e 's/__STG_SKILL_NAME__-/stg-/g' "$file"
                ;;
            *)
                sed -i '' -E "s/(^|[^a-zA-Z0-9])stg([^a-zA-Z0-9]|$)/\1${NEW_PROJECT_NAME}\2/g" "$file"
                ;;
        esac
    fi
done

# Исключение папок, которые не нужно копировать (git, vendor, deploy)
if [ -d "./.git" ]; then
    rm -rf "./.git"
    echo "Удалена папка .git"
fi

if [ -d "./vendor" ]; then
    rm -rf "${NEW_DIR}/vendor"
    echo "Удалена папка vendor"
fi

if [ -d "./deploy" ]; then
    rm -rf "${NEW_DIR}/deploy"
    echo "Удалена папка deploy"
fi


go mod tidy

make generate
# Инициализация нового git репозитория
git init
git add .
git commit -m "Initial commit"

echo ""
echo "Проект успешно создан в: $NEW_DIR"
echo "Имя проекта: $NEW_PROJECT_NAME"
echo "Все вхождения 'stg' заменены на '$NEW_PROJECT_NAME'"
