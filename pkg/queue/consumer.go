package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"stg/pkg/connections/rediscli"
)

type consumer struct {
	redis  RedisClient
	group  string
	config Settings
	logger *zap.Logger
	prefix string
}

type pendingScan struct {
	cursor      string
	nextReclaim time.Time
}

func (cg *consumer) consume(ctx context.Context, stream string, callback ConsumerCallback) error {
	cg.consumeStream(ctx, stream, uuid.NewString(), callback)

	return ctx.Err()
}

func (cg *consumer) createGroup(ctx context.Context, stream string) error {
	err := cg.redis.Call(ctx, "queue_create_group", func(ctx context.Context, cli redis.UniversalClient, kf rediscli.KeyFormatter) error {
		err := cli.XGroupCreateMkStream(ctx, streamKey(kf, cg.prefix, stream), cg.group, cg.config.StartID).Err()
		if err != nil && strings.HasPrefix(err.Error(), "BUSYGROUP ") {
			return nil
		}

		return err
	})
	if err != nil {
		return fmt.Errorf("create group %s on %s: %w", cg.group, stream, err)
	}

	return nil
}

func (cg *consumer) consumeStream(ctx context.Context, stream, consumer string, callback ConsumerCallback) {
	slots := make(chan struct{}, cg.config.Concurrency)
	var handlers sync.WaitGroup
	defer handlers.Wait()
	scan := pendingScan{cursor: "0-0"}
	for ctx.Err() == nil {
		capacity := cg.reserve(ctx, slots)
		if capacity == 0 {
			return
		}
		entries, err := cg.fetch(ctx, stream, consumer, int64(capacity), &scan)
		for i := len(entries); i < capacity; i++ {
			<-slots
		}
		for _, entry := range entries {
			handlers.Go(func() {
				defer func() { <-slots }()
				cg.handle(ctx, stream, consumer, entry, callback)
			})
		}
		if err != nil && ctx.Err() == nil {
			cg.report("read", stream, err)
			if !wait(ctx, 500*time.Millisecond) {
				return
			}
		}
	}
}

func (cg *consumer) fetch(ctx context.Context, stream, name string, count int64, scan *pendingScan) ([]redis.XMessage, error) {
	if !time.Now().Before(scan.nextReclaim) {
		entries, cursor, err := cg.reclaim(ctx, stream, name, scan.cursor, count)
		scan.cursor = cursor
		if cursor == "0-0" || err != nil {
			scan.nextReclaim = time.Now().Add(cg.config.ReclaimInterval)
		}
		if err != nil || len(entries) > 0 {
			return entries, err
		}
	}

	return cg.read(ctx, stream, name, count)
}

// Reserve processing slots before fetching, so prefetched messages cannot expire in a local backlog.
func (cg *consumer) reserve(ctx context.Context, slots chan struct{}) int {
	select {
	case <-ctx.Done():
		return 0
	case slots <- struct{}{}:
	}
	n := 1
	for n < cg.config.BatchSize {
		select {
		case slots <- struct{}{}:
			n++
		default:
			return n
		}
	}

	return n
}

func (cg *consumer) read(ctx context.Context, stream, consumer string, count int64) ([]redis.XMessage, error) {
	var entries []redis.XMessage
	err := cg.redis.Call(ctx, "queue_read", func(ctx context.Context, cli redis.UniversalClient, kf rediscli.KeyFormatter) error {
		streams, err := cli.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: cg.group, Consumer: consumer, Streams: []string{streamKey(kf, cg.prefix, stream), ">"},
			Count: count, Block: cg.config.BlockTimeout,
		}).Result()
		if errors.Is(err, redis.Nil) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, result := range streams {
			entries = append(entries, result.Messages...)
		}

		return nil
	})

	return entries, err
}

func (cg *consumer) reclaim(ctx context.Context, stream, consumer, start string, count int64) ([]redis.XMessage, string, error) {
	var entries []redis.XMessage
	next := start
	err := cg.redis.Call(ctx, "queue_reclaim", func(ctx context.Context, cli redis.UniversalClient, kf rediscli.KeyFormatter) error {
		var err error
		entries, next, err = cli.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream: streamKey(kf, cg.prefix, stream), Group: cg.group, Consumer: consumer,
			MinIdle: cg.config.ReclaimIdle, Start: start, Count: count,
		}).Result()

		return err
	})
	if err == nil {
		reclaimedTotal.WithLabelValues(stream, cg.group).Add(float64(len(entries)))
	}

	return entries, next, err
}

func (cg *consumer) report(operation, stream string, err error) {
	consumerErrors.WithLabelValues(stream, cg.group, operation).Inc()
	err = fmt.Errorf("queue %s, stream %s, group %s: %w", operation, stream, cg.group, err)
	cg.logger.Warn("queue error", zap.Error(err))
}

func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
