package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

var (
	// ErrPoolNotFound happens when try to get non existing pool.
	ErrPoolNotFound = errors.New("pool not found")
)

type Pool struct {
	cfgs  PoolConfig
	pools map[string]*Postgres
	mu    sync.RWMutex

	logger *zap.Logger

	beforeConnectCallbacks []BeforeConnectCallback
	afterConnectCallbacks  []AfterConnectCallback
}

func NewPostgresPool(params PoolParams) (*Pool, error) {
	p := &Pool{
		cfgs:  params.Cfgs,
		pools: make(map[string]*Postgres, len(params.Cfgs)),

		logger: params.Logger,

		beforeConnectCallbacks: params.BeforeConnectCallbacks,
		afterConnectCallbacks:  params.AfterConnectCallbacks,
	}

	for name, cfg := range p.cfgs {
		pg := NewPostgres(
			cfg,
			p.logger,
			p.beforeConnectCallbacks,
			p.afterConnectCallbacks,
		)

		if err := pg.Start(context.Background()); err != nil {
			p.logger.Error("start postgress", zap.Error(err))

			return nil, fmt.Errorf("start postgress conntect: %w", err)
		}

		p.pools[name] = pg
	}

	return p, nil
}

func (p *Pool) GetPool(name string) (*Postgres, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if pool, ok := p.pools[name]; ok {
		return pool, nil
	}

	return nil, ErrPoolNotFound
}

func (p *Pool) Stop(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, pool := range p.pools {
		err := pool.Stop(ctx)
		if err != nil {
			p.logger.Error("stop postgress", zap.Error(err))

			return err
		}
	}

	return nil
}
