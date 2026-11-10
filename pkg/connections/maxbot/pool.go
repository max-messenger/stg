package maxbot

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/max-messenger/maxbot"
	"github.com/max-messenger/maxbot/middleware"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

var ErrPoolNotFound = errors.New("pool not found")

type PoolConfig map[string]Config

type Config struct {
	ApiUrl string `yaml:"api_url"` // nolint:all
	Token  string `yaml:"token"`
	URL    string `yaml:"url"`
	Path   string `yaml:"path"`
	Secret string `yaml:"secret"` // nolint:gosec
}

type Pool struct {
	configs PoolConfig
	pools   map[string]*maxbot.Api
}

func NewMaxBotPool(params PoolParams, log *zap.Logger) (*Pool, error) {
	p := &Pool{
		configs: params.Configs,
		pools:   make(map[string]*maxbot.Api, len(params.Configs)),
	}

	for name, cfg := range p.configs {
		if cfg.ApiUrl != "" {
			params.Opts = append(params.Opts, maxbot.WithBaseURL(cfg.ApiUrl))
		}

		api, err := maxbot.NewApi(cfg.Token, params.Opts...)
		if err != nil {
			return nil, err
		}

		for _, mr := range params.Middlewares {
			api.Use(mr)
		}

		api.Use(middleware.Recover(func(err error, ctx maxbot.Context) {
			log.Error(fmt.Sprintf("message recover: %v", ctx), zap.Error(err))
		}))
		api.Use(func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
			return func(c maxbot.Context) error {
				_, span := otel.Tracer("maxbot").Start(c.Context(), "bot-message")
				defer span.End()

				defer func(t time.Time) {
					metricRequestsTotal.WithLabelValues("bot-message").Inc()
					metricRequestsDuration.WithLabelValues("bot-message").Observe(time.Since(t).Seconds())
				}(time.Now())

				return next(c)
			}
		})
		go api.Start()

		p.pools[name] = api
	}

	return p, nil
}

func (p *Pool) GetPool(name string) (*maxbot.Api, error) {
	if pool, ok := p.pools[name]; ok {
		return pool, nil
	}

	return nil, ErrPoolNotFound
}

func (p *Pool) Register(r chi.Router) {
	for name, cfg := range p.configs {
		r.Handle(cfg.Path, p.pools[name].WebhookHandler())
	}
}
