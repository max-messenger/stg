// Package queue provides a shared Redis Streams queue with callback-based consumption.
package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"stg/pkg/connections/rediscli"
)

var (
	ErrOwnershipLost = errors.New("message ownership lost")
)

// Queue publishes and consumes streams using one configured Redis connection.
// It does not own the connection; callbacks and Consume calls may run concurrently.
type Queue struct {
	redis  RedisClient
	config Config
	logger *zap.Logger
}

func New(cfg Config, logger *zap.Logger, client RedisClient) (*Queue, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	cfg, err := cfg.Prepare()
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Queue{redis: client, config: cfg, logger: logger}, nil
}

func (q *Queue) queueConfig(name string) Settings {
	if cfg, ok := q.config.Queues[name]; ok {
		return cfg
	}

	return q.config.Defaults
}

// Consume blocks until cancellation and waits for active callbacks to finish.
// Calls for the same stream and group share messages; each call has its own concurrency limit.
func (q *Queue) Consume(ctx context.Context, stream string, callback ConsumerCallback) error {
	if stream == "" || callback == nil {
		return fmt.Errorf("stream and callback are required")
	}
	cfg := q.queueConfig(stream)
	c := &consumer{
		redis: q.redis, group: cfg.Group, config: cfg, logger: q.logger,
		prefix: q.config.Redis.Prefix,
	}
	if err := c.createGroup(ctx, stream); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	return c.consume(ctx, stream, callback)
}

// Message contains a binary payload and metadata assigned by Redis.
type Message struct {
	ID        string
	Stream    string
	Payload   []byte
	Timestamp time.Time
}

type ConsumerCallback func(context.Context, Message) error

// RedisClient is implemented by rediscli.Redis.
type RedisClient interface {
	Call(context.Context, string, rediscli.CallCallback) error
}

func streamKey(kf rediscli.KeyFormatter, prefix, name string) string {
	if prefix != "" {
		name = prefix + ":" + name
	}

	return kf.FormatKey(name)
}
