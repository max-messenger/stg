package authctrl

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"stg/internal/app/domain"
	"stg/internal/app/services/auth"
)

// AuthService интерфейс auth-сервиса.
type AuthService interface {
	Authenticate(ctx context.Context, initData string) (domain.AuthData, error)
	Refresh(ctx context.Context, refreshToken string) (domain.AuthData, error)
	Logout(ctx context.Context, refreshToken string) error
	ParseAndValidateToken(ctx context.Context, token string, tokenType string) (domain.AuthData, error)
}

// Controller обрабатывает запросы авторизации.
type Controller struct {
	logger     *zap.Logger
	srv        AuthService
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewController создаёт экземпляр Controller.
func NewController(l *zap.Logger, s AuthService, cfg auth.Config) *Controller {
	return &Controller{
		logger:     l,
		srv:        s,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
	}
}

// Register регистрирует маршруты.
func (c *Controller) Register(r chi.Router) {
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/login", c.loginHandler)
		r.Post("/refresh", c.refreshHandler)
		r.Post("/logout", c.logoutHandler)
		r.Get("/me", c.meHandler)
	})
}
