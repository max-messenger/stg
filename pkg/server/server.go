package server

import (
	"context"

	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

// Controller api controller.
type Controller interface {
	Register(r chi.Router)
}

type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type ServersParams struct {
	fx.In

	HTTPServer   *HTTPServer `optional:"true"`
	GRPCServer   *GRPCServer `optional:"true"`
	SystemServer *SystemServer
}
type Servers struct {
	servers []Server
}

func NewServersProvider(params ServersParams) *Servers {
	servers := make([]Server, 0, 3)
	if params.HTTPServer != nil {
		servers = append(servers, params.HTTPServer)
	}

	if params.GRPCServer != nil {
		servers = append(servers, params.GRPCServer)
	}

	servers = append(servers, params.SystemServer)

	return &Servers{servers: servers}

}

func (s *Servers) Start(ctx context.Context) error {
	for _, srv := range s.servers {
		if err := srv.Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *Servers) Stop(ctx context.Context) error {
	for _, srv := range s.servers {
		if err := srv.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}
