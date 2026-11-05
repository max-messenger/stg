package queue

import (
	"fmt"

	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/pkg/connections/rediscli"
)

var Module = fx.Module(
	"queue",
	fx.Provide(
		configProvider("queue"),
		redisAdapter,
		New,
	),
	fx.Decorate(func(log *zap.Logger) *zap.Logger {
		return log.Named("queue")
	}),
)

func configProvider(name string) func(config.Provider) (Config, error) {
	return func(cp config.Provider) (Config, error) {
		cfg := NewConfig()
		if err := cp.Get(name).Populate(&cfg); err != nil {
			return Config{}, err
		}

		return cfg, nil
	}
}

func redisAdapter(cfg Config, pool *rediscli.Pool) (RedisClient, error) {
	if cfg.Redis.PoolName == "" {
		return nil, fmt.Errorf("queue.redis.pool_name is required")
	}
	rd, err := pool.GetPool(cfg.Redis.PoolName)
	if err != nil {
		return nil, fmt.Errorf("queue redis pool %q: %w", cfg.Redis.PoolName, err)
	}

	return rd, nil
}
