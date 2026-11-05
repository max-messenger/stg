package queue_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"stg/pkg/connections/rediscli"
	"stg/pkg/queue"
)

func integrationRedis(t *testing.T) (*redis.Client, *rediscli.Redis, string) {
	t.Helper()
	addr := os.Getenv("REDIS_SEED")
	if addr == "" {
		t.Skip("set REDIS_SEED to run Redis integration tests")
	}
	client := redis.NewClient(&redis.Options{
		Addr: addr, ContextTimeoutEnabled: true, MaxRetries: -1,
	})
	stream := "gibs:queue:test:" + uuid.NewString()
	t.Cleanup(func() {
		require.NoError(t, client.Del(context.Background(), stream, stream+":second").Err())
		require.NoError(t, client.Close())
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, client.Ping(ctx).Err())
	rd, err := rediscli.NewRedis(zap.NewNop(), &rediscli.Config{
		Addrs: []string{addr}, MaxRetries: -1,
	}, "queue-test")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rd.Close()) })

	return client, rd, stream
}

func testConfig() queue.Config {
	return queue.Config{Defaults: queue.Settings{
		BlockTimeout: 10 * time.Millisecond, ReclaimInterval: 20 * time.Millisecond,
		ReclaimIdle: 300 * time.Millisecond,
	}}
}

func startConsumer(t *testing.T, q *queue.Queue, stream string, handler queue.ConsumerCallback) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- q.Consume(ctx, stream, handler) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			select {
			case err := <-done:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(5 * time.Second):
				t.Error("consumer did not stop")
			}
		})
	}
	t.Cleanup(stop)
	return stop
}

func pending(t *testing.T, client *redis.Client, stream, group string) int64 {
	t.Helper()
	pel, err := client.XPending(context.Background(), stream, group).Result()
	require.NoError(t, err)
	return pel.Count
}

func TestIntegration_QueueAckModes(t *testing.T) {
	for _, mode := range []queue.AckMode{queue.AckOnSuccess, queue.AckBeforeHandle, queue.AckAfterHandle} {
		for _, fails := range []bool{false, true} {
			t.Run(string(mode)+map[bool]string{false: "/success", true: "/error"}[fails], func(t *testing.T) {
				client, rd, stream := integrationRedis(t)
				producer, err := queue.New(queue.Config{}, zap.NewNop(), rd)
				require.NoError(t, err)
				id, err := producer.Publish(context.Background(), stream, []byte{0, 255, 'x'})
				require.NoError(t, err)
				cfg := testConfig()
				cfg.Defaults.AckMode, cfg.Defaults.ReclaimIdle = mode, time.Minute
				consumer, err := queue.New(cfg, zap.NewNop(), rd)
				require.NoError(t, err)
				entered, release, returned := make(chan queue.Message, 1), make(chan struct{}), make(chan struct{})
				handler := func(ctx context.Context, msg queue.Message) error {
					entered <- msg
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-release:
					}
					defer close(returned)
					if fails {
						return errors.New("business error")
					}
					return nil
				}
				stop := startConsumer(t, consumer, stream, handler)
				select {
				case msg := <-entered:
					require.Equal(t, id, msg.ID)
					require.Equal(t, stream, msg.Stream)
					require.Equal(t, []byte{0, 255, 'x'}, msg.Payload)
					require.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
				case <-time.After(5 * time.Second):
					t.Fatal("callback was not called")
				}
				if mode == queue.AckBeforeHandle {
					require.Zero(t, pending(t, client, stream, "workers"))
				} else {
					require.EqualValues(t, 1, pending(t, client, stream, "workers"))
				}
				close(release)
				<-returned
				if !fails || mode != queue.AckOnSuccess {
					require.Eventually(t, func() bool { return pending(t, client, stream, "workers") == 0 },
						5*time.Second, 10*time.Millisecond)
				}
				stop()
				if fails && mode == queue.AckOnSuccess {
					require.EqualValues(t, 1, pending(t, client, stream, "workers"))
				}
			})
		}
	}
}

func TestIntegration_QueueRecoverAbandoned(t *testing.T) {
	client, rd, stream := integrationRedis(t)
	ctx := context.Background()
	producer, err := queue.New(queue.Config{}, zap.NewNop(), rd)
	require.NoError(t, err)
	require.NoError(t, producer.Send(ctx, stream, []byte("recover")))
	require.NoError(t, client.XGroupCreate(ctx, stream, "workers", "0").Err())
	_, err = client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: "workers", Consumer: "dead-process", Streams: []string{stream, ">"}, Count: 1, Block: -1,
	}).Result()
	require.NoError(t, err)
	consumer, err := queue.New(testConfig(), zap.NewNop(), rd)
	require.NoError(t, err)
	var called atomic.Int32
	handler := func(context.Context, queue.Message) error {
		called.Add(1)
		return nil
	}
	stop := startConsumer(t, consumer, stream, handler)
	require.Eventually(t, func() bool {
		return called.Load() == 1 && pending(t, client, stream, "workers") == 0
	}, 5*time.Second, 10*time.Millisecond)
	stop()
}

func TestIntegration_QueueRetryCallback(t *testing.T) {
	client, rd, stream := integrationRedis(t)
	producer, err := queue.New(queue.Config{}, zap.NewNop(), rd)
	require.NoError(t, err)
	require.NoError(t, producer.Send(context.Background(), stream, []byte("retry")))
	consumer, err := queue.New(testConfig(), zap.NewNop(), rd)
	require.NoError(t, err)
	var called atomic.Int32
	handler := func(context.Context, queue.Message) error {
		if called.Add(1) == 1 {
			return errors.New("temporary business error")
		}
		return nil
	}
	stop := startConsumer(t, consumer, stream, handler)
	require.Eventually(t, func() bool {
		return called.Load() == 2 && pending(t, client, stream, "workers") == 0
	}, 5*time.Second, 10*time.Millisecond)
	stop()
}

func TestIntegration_QueueActiveHandlerNotReclaimed(t *testing.T) {
	client, rd, stream := integrationRedis(t)
	producer, err := queue.New(queue.Config{}, zap.NewNop(), rd)
	require.NoError(t, err)
	require.NoError(t, producer.Send(context.Background(), stream, []byte("slow")))
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var called atomic.Int32
	handler := func(ctx context.Context, _ queue.Message) error {
		called.Add(1)
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			return nil
		}
	}
	first, err := queue.New(testConfig(), zap.NewNop(), rd)
	require.NoError(t, err)
	stopFirst := startConsumer(t, first, stream, handler)
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("callback was not called")
	}
	second, err := queue.New(testConfig(), zap.NewNop(), rd)
	require.NoError(t, err)
	stopSecond := startConsumer(t, second, stream, handler)
	// Several reclaim windows pass while the first callback remains active.
	time.Sleep(time.Second)
	require.EqualValues(t, 1, called.Load())
	close(release)
	require.Eventually(t, func() bool { return pending(t, client, stream, "workers") == 0 },
		5*time.Second, 10*time.Millisecond)
	stopFirst()
	stopSecond()
}

func TestIntegration_QueueGroupsAndMultipleStreams(t *testing.T) {
	_, rd, stream := integrationRedis(t)
	producer, err := queue.New(queue.Config{}, zap.NewNop(), rd)
	require.NoError(t, err)
	for _, name := range []string{stream, stream + ":second"} {
		require.NoError(t, producer.Send(context.Background(), name, []byte("event")))
	}
	var called atomic.Int32
	for _, group := range []string{"first", "second"} {
		cfg := testConfig()
		cfg.Defaults.Group = group
		consumer, err := queue.New(cfg, zap.NewNop(), rd)
		require.NoError(t, err)
		for _, name := range []string{stream, stream + ":second"} {
			handler := func(context.Context, queue.Message) error {
				called.Add(1)
				return nil
			}
			startConsumer(t, consumer, name, handler)
		}
	}
	require.Eventually(t, func() bool { return called.Load() == 4 }, 5*time.Second, 10*time.Millisecond)
}

func TestIntegration_QueueConcurrencyAndMetrics(t *testing.T) {
	client, rd, stream := integrationRedis(t)
	producer, err := queue.New(queue.Config{}, zap.NewNop(), rd)
	require.NoError(t, err)
	for range 20 {
		require.NoError(t, producer.Send(context.Background(), stream, []byte("event")))
	}
	cfg := testConfig()
	cfg.Defaults.Concurrency, cfg.Defaults.BatchSize = 3, 10
	consumer, err := queue.New(cfg, zap.NewNop(), rd)
	require.NoError(t, err)
	var active, peak, completed atomic.Int32
	handler := func(ctx context.Context, _ queue.Message) error {
		current := active.Add(1)
		defer active.Add(-1)
		for previous := peak.Load(); current > previous; previous = peak.Load() {
			if peak.CompareAndSwap(previous, current) {
				break
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Millisecond):
			completed.Add(1)
			return nil
		}
	}
	stop := startConsumer(t, consumer, stream, handler)
	require.Eventually(t, func() bool {
		return completed.Load() == 20 && pending(t, client, stream, "workers") == 0
	}, 5*time.Second, 10*time.Millisecond)
	require.EqualValues(t, 3, peak.Load())
	require.Eventually(t, func() bool {
		families, err := prometheus.DefaultGatherer.Gather()
		require.NoError(t, err)
		for _, family := range families {
			if family.GetName() != "redis_queue_processed_total" {
				continue
			}
			for _, metric := range family.Metric {
				for _, label := range metric.Label {
					if label.GetName() == "stream" && label.GetValue() == stream && metric.GetCounter().GetValue() == 20 {
						return true
					}
				}
			}
		}
		return false
	}, 5*time.Second, 10*time.Millisecond)
	stop()
}

func TestIntegration_QueueTimeoutAndOwnershipLoss(t *testing.T) {
	for _, scenario := range []string{"timeout", "ownership_loss"} {
		t.Run(scenario, func(t *testing.T) {
			client, rd, stream := integrationRedis(t)
			cfg := testConfig()
			if scenario == "timeout" {
				cfg.Defaults.HandlerTimeout = 100 * time.Millisecond
			}
			q, err := queue.New(cfg, zap.NewNop(), rd)
			require.NoError(t, err)
			id, err := q.Publish(context.Background(), stream, []byte("event"))
			require.NoError(t, err)
			entered, exited := make(chan struct{}), make(chan struct{})
			handler := func(ctx context.Context, _ queue.Message) error {
				close(entered)
				<-ctx.Done()
				close(exited)
				return ctx.Err()
			}
			stop := startConsumer(t, q, stream, handler)
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("callback was not called")
			}
			if scenario == "ownership_loss" {
				_, err = client.XClaimJustID(context.Background(), &redis.XClaimArgs{
					Stream: stream, Group: "workers", Consumer: "other-process", Messages: []string{id},
				}).Result()
				require.NoError(t, err)
			}
			select {
			case <-exited:
			case <-time.After(5 * time.Second):
				t.Fatal("callback context was not canceled")
			}
			stop()
			require.EqualValues(t, 1, pending(t, client, stream, "workers"))
		})
	}
}

func TestIntegration_QueueFxSharedConnection(t *testing.T) {
	client, _, stream := integrationRedis(t)
	prefix := "queue-fx"
	key := prefix + ":notifications:" + stream
	t.Cleanup(func() {
		require.NoError(t, client.Del(context.Background(), key).Err())
	})
	provider, err := config.NewYAML(config.Static(map[string]any{
		"redis": map[string]any{
			"main": map[string]any{"addrs": []string{os.Getenv("REDIS_SEED")}, "key_prefix": prefix},
		},
		"queue": map[string]any{
			"redis":    map[string]any{"pool_name": "main", "prefix": "notifications"},
			"defaults": map[string]any{"group": "shared", "block_timeout": "10ms"},
		},
	}))
	require.NoError(t, err)
	var q *queue.Queue
	app := fx.New(
		fx.NopLogger,
		fx.Provide(func() config.Provider { return provider }, zap.NewNop),
		rediscli.Module, queue.Module, fx.Populate(&q),
	)
	require.NoError(t, app.Err())
	require.NoError(t, app.Start(context.Background()))
	t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
	require.NoError(t, q.Send(context.Background(), stream, []byte("shared connection")))
	exists, err := client.Exists(context.Background(), key).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, exists)
	var called atomic.Bool
	stop := startConsumer(t, q, stream, func(_ context.Context, msg queue.Message) error {
		called.Store(string(msg.Payload) == "shared connection")
		return nil
	})
	require.Eventually(t, func() bool {
		return called.Load() && pending(t, client, key, "shared") == 0
	}, 5*time.Second, 10*time.Millisecond)
	stop()
}

func TestIntegration_QueueNamedSettings(t *testing.T) {
	client, rd, stream := integrationRedis(t)
	cfg := testConfig()
	cfg.Redis.Prefix = "named"
	cfg.Queues = map[string]queue.Settings{
		stream:             {Group: "early", AckMode: queue.AckBeforeHandle, ReclaimIdle: time.Minute},
		stream + ":second": {Group: "late", ReclaimIdle: time.Minute},
	}
	q, err := queue.New(cfg, zap.NewNop(), rd)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Del(context.Background(), "named:"+stream, "named:"+stream+":second").Err())
	})
	for _, name := range []string{stream, stream + ":second"} {
		require.NoError(t, q.Send(context.Background(), name, []byte("event")))
		entered := make(chan string, 1)
		stop := startConsumer(t, q, name, func(_ context.Context, msg queue.Message) error {
			entered <- msg.Stream
			return errors.New("business error")
		})
		select {
		case received := <-entered:
			require.Equal(t, name, received)
		case <-time.After(5 * time.Second):
			t.Fatal("callback was not called")
		}
		stop()
		if name == stream {
			require.Zero(t, pending(t, client, "named:"+name, "early"))
		} else {
			require.EqualValues(t, 1, pending(t, client, "named:"+name, "late"))
		}
	}
}
