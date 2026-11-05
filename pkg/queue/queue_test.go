package queue_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"stg/pkg/connections/rediscli"
	"stg/pkg/queue"
)

func TestUnit_Config(t *testing.T) {
	cfg, err := (queue.Config{}).Prepare()
	require.NoError(t, err)
	require.Equal(t, queue.AckOnSuccess, cfg.Defaults.AckMode)
	require.Positive(t, cfg.Defaults.Concurrency)
	require.Positive(t, cfg.Defaults.BlockTimeout)
	for _, invalid := range []queue.Settings{
		{AckMode: "unknown"}, {Concurrency: -1}, {BatchSize: -1},
		{BlockTimeout: -1}, {ReclaimInterval: -1}, {ReclaimIdle: time.Millisecond},
	} {
		_, err = (queue.Config{Defaults: invalid}).Prepare()
		require.Error(t, err)
	}
}

func TestUnit_Send(t *testing.T) {
	rd := newTestRedis()
	p, err := queue.New(queue.Config{}, zap.NewNop(), rd)
	require.NoError(t, err)
	payload := []byte{0, 255, 'x'}
	require.NoError(t, p.Send(context.Background(), "events", payload))
	rd.mu.Lock()
	defer rd.mu.Unlock()
	require.Equal(t, "test:events", rd.stream)
	require.Equal(t, string(payload), rd.entries[0].Values["payload"])
}

func TestUnit_AckModes(t *testing.T) {
	for _, mode := range []queue.AckMode{queue.AckOnSuccess, queue.AckBeforeHandle, queue.AckAfterHandle} {
		for _, fails := range []bool{false, true} {
			t.Run(string(mode)+map[bool]string{false: "/success", true: "/error"}[fails], func(t *testing.T) {
				rd := newTestRedis()
				p, err := queue.New(queue.Config{}, zap.NewNop(), rd)
				require.NoError(t, err)
				require.NoError(t, p.Send(context.Background(), "events", []byte("hello")))
				cg, err := queue.New(queue.Config{Defaults: queue.Settings{AckMode: mode}}, zap.NewNop(), rd)
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(context.Background())
				t.Cleanup(cancel)
				var ackInside bool
				var payload string
				handler := func(_ context.Context, msg queue.Message) error {
					payload = string(msg.Payload)
					rd.mu.Lock()
					ackInside = rd.acked
					rd.mu.Unlock()
					cancel()
					if fails {
						return errors.New("handler failed")
					}
					return nil
				}
				done := make(chan error, 1)
				go func() { done <- cg.Consume(ctx, "events", handler) }()
				require.ErrorIs(t, waitConsumer(t, done), context.Canceled)
				require.Equal(t, "hello", payload)
				require.Equal(t, mode == queue.AckBeforeHandle, ackInside)
				rd.mu.Lock()
				defer rd.mu.Unlock()
				// Cancellation leaves late-ACK messages pending even if the callback returns nil.
				require.Equal(t, mode == queue.AckBeforeHandle, rd.acked)
			})
		}
	}
}

func TestUnit_HandlerResult(t *testing.T) {
	for _, mode := range []queue.AckMode{queue.AckOnSuccess, queue.AckBeforeHandle, queue.AckAfterHandle} {
		t.Run(string(mode), func(t *testing.T) {
			rd := newTestRedis()
			p, err := queue.New(queue.Config{}, zap.NewNop(), rd)
			require.NoError(t, err)
			require.NoError(t, p.Send(context.Background(), "events", []byte("hello")))
			cg, err := queue.New(queue.Config{Defaults: queue.Settings{AckMode: mode}}, zap.NewNop(), rd)
			require.NoError(t, err)
			called := make(chan struct{}, 1)
			handler := func(context.Context, queue.Message) error {
				called <- struct{}{}
				return errors.New("failed")
			}
			stop := startConsumer(t, cg, "events", handler)
			select {
			case <-called:
			case <-time.After(time.Second):
				t.Fatal("callback was not called")
			}
			if mode == queue.AckOnSuccess {
				require.Eventually(t, func() bool { return rd.hasPending() }, time.Second, time.Millisecond)
			} else {
				require.Eventually(t, rd.isAcked, time.Second, time.Millisecond)
			}
			stop()
			require.Equal(t, mode != queue.AckOnSuccess, rd.isAcked())
		})
	}
}

func TestUnit_PendingRedeliveryAndPanic(t *testing.T) {
	rd := newTestRedis()
	p, err := queue.New(queue.Config{}, zap.NewNop(), rd)
	require.NoError(t, err)
	require.NoError(t, p.Send(context.Background(), "events", []byte("hello")))
	cg, err := queue.New(queue.Config{Defaults: queue.Settings{
		BlockTimeout: 5 * time.Millisecond, ReclaimInterval: 10 * time.Millisecond, ReclaimIdle: 30 * time.Millisecond,
	}}, zap.NewNop(), rd)
	require.NoError(t, err)
	var attempts atomic.Int32
	handler := func(context.Context, queue.Message) error {
		if attempts.Add(1) == 1 {
			panic("handler panic")
		}
		return nil
	}
	stop := startConsumer(t, cg, "events", handler)
	require.Eventually(t, rd.isAcked, time.Second, time.Millisecond)
	stop()
	require.EqualValues(t, 2, attempts.Load())
}

func TestUnit_QueueValidation(t *testing.T) {
	_, err := queue.New(queue.Config{}, zap.NewNop(), nil)
	require.Error(t, err)
	cg, err := queue.New(queue.Config{}, zap.NewNop(), newTestRedis())
	require.NoError(t, err)
	handler := func(context.Context, queue.Message) error { return nil }
	require.Error(t, cg.Consume(context.Background(), "", handler))
	require.Error(t, cg.Consume(context.Background(), "events", nil))
	require.Error(t, cg.Send(context.Background(), "", nil))
}

func waitConsumer(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal("consumer did not finish")
		return nil
	}
}

func TestUnit_RedisCallOutcomes(t *testing.T) {
	transportErr := errors.New("redis unavailable")
	for _, tc := range []struct {
		name      string
		operation string
		groupErr  error
		readErr   error
		wantErr   error
	}{
		{name: "existing group", operation: "queue_create_group", groupErr: errors.New("BUSYGROUP Consumer Group name already exists")},
		{name: "empty read", operation: "queue_read"},
		{name: "group failure", operation: "queue_create_group", groupErr: transportErr, wantErr: transportErr},
		{name: "read failure", operation: "queue_read", readErr: transportErr, wantErr: transportErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rd := newTestRedis()
			rd.groupErr, rd.readErr = tc.groupErr, tc.readErr
			rd.observedOperation, rd.observedErrors = tc.operation, make(chan error, 1)
			q, err := queue.New(queue.Config{}, zap.NewNop(), rd)
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			done := make(chan error, 1)
			go func() { done <- q.Consume(ctx, "events", func(context.Context, queue.Message) error { return nil }) }()
			var observed error
			select {
			case observed = <-rd.observedErrors:
			case <-time.After(time.Second):
				t.Fatal("Redis operation was not called")
			}
			cancel()
			consumeErr := waitConsumer(t, done)
			if tc.operation == "queue_create_group" && tc.wantErr != nil {
				require.ErrorIs(t, consumeErr, tc.wantErr)
			} else {
				require.ErrorIs(t, consumeErr, context.Canceled)
			}
			if tc.wantErr == nil {
				require.NoError(t, observed)
			} else {
				require.ErrorIs(t, observed, tc.wantErr)
			}
		})
	}
}

// testRedis models stream delivery and the PEL, while callbacks and concurrency run normally.
type testRedis struct {
	redis.UniversalClient
	mu                sync.Mutex
	stream            string
	entries           []redis.XMessage
	read              bool
	owner             string
	idle              time.Time
	acked             bool
	groupErr          error
	readErr           error
	observedOperation string
	observedErrors    chan error
}

func newTestRedis() *testRedis { return &testRedis{} }

func (r *testRedis) FormatKey(key string) string { return "test:" + key }

func (r *testRedis) Call(ctx context.Context, operation string, f rediscli.CallCallback) error {
	err := f(ctx, r, r)
	if operation == r.observedOperation {
		select {
		case r.observedErrors <- err:
		default:
		}
	}
	return err
}

func (r *testRedis) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stream = args.Stream
	r.entries = append(r.entries, redis.XMessage{ID: "1-0", Values: args.Values.(map[string]any)})
	return redis.NewStringResult("1-0", nil)
}

func (r *testRedis) XGroupCreateMkStream(context.Context, string, string, string) *redis.StatusCmd {
	return redis.NewStatusResult("OK", r.groupErr)
}

func (r *testRedis) XReadGroup(ctx context.Context, args *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	if r.readErr != nil {
		cmd := redis.NewXStreamSliceCmd(ctx)
		cmd.SetErr(r.readErr)
		return cmd
	}
	r.mu.Lock()
	if !r.read && len(r.entries) > 0 {
		r.read, r.owner, r.idle = true, args.Consumer, time.Now()
		entries := append([]redis.XMessage(nil), r.entries...)
		r.mu.Unlock()
		cmd := redis.NewXStreamSliceCmd(ctx)
		cmd.SetVal([]redis.XStream{{Stream: args.Streams[0], Messages: entries}})
		return cmd
	}
	r.mu.Unlock()
	cmd := redis.NewXStreamSliceCmd(ctx)
	select {
	case <-ctx.Done():
		cmd.SetErr(ctx.Err())
	case <-time.After(5 * time.Millisecond):
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (r *testRedis) XAutoClaim(ctx context.Context, args *redis.XAutoClaimArgs) *redis.XAutoClaimCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	var entries []redis.XMessage
	if r.read && !r.acked && time.Since(r.idle) >= args.MinIdle {
		entries = append(entries, r.entries...)
		r.owner, r.idle = args.Consumer, time.Now()
	}
	cmd := redis.NewXAutoClaimCmd(ctx)
	cmd.SetVal(entries, "0-0")
	return cmd
}

func (r *testRedis) EvalSha(ctx context.Context, _ string, _ []string, args ...any) *redis.Cmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	cmd := redis.NewCmd(ctx)
	if r.acked || r.owner != args[1] {
		cmd.SetVal(int64(0))
		return cmd
	}
	if args[3] == "ack" {
		r.acked = true
	} else {
		r.idle = time.Now()
	}
	cmd.SetVal(int64(1))
	return cmd
}

func (r *testRedis) isAcked() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.acked
}

func (r *testRedis) hasPending() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.read && !r.acked
}
