package bgtasker_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"stg/pkg/bgtasker"
)

func TestUnitRunnerStopIsIdempotent(t *testing.T) {
	r := bgtasker.NewTasker(zap.NewNop(), bgtasker.DefaultTaskConfig(), t.Name())
	require.NoError(t, r.Start(t.Context()))
	require.NoError(t, r.Stop(t.Context()))
	require.NotPanics(t, func() { require.NoError(t, r.Stop(t.Context())) })
	require.NotPanics(t, func() {
		r.Run(t.Context(), func(context.Context) error {
			t.Error("task executed after stop")
			return nil
		})
	})
}

func TestUnitRunnerShutdownDrainsQueue(t *testing.T) {
	for _, mode := range []string{bgtasker.RunnerTypeBlock, bgtasker.RunnerTypeDrop} {
		t.Run(mode, func(t *testing.T) {
			r := bgtasker.NewTasker(zap.NewNop(), &bgtasker.TaskConfig{
				RunnerType: mode, AsyncWorkersNum: 1, AsyncBufferSize: 1, AsyncWorkerTimeout: time.Second,
			}, t.Name())
			require.NoError(t, r.Start(t.Context()))
			started := make(chan struct{})
			release := make(chan struct{})
			var releaseOnce sync.Once
			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })
				require.NoError(t, r.Stop(context.Background()))
			})
			var executed atomic.Int64
			r.Run(t.Context(), func(context.Context) error {
				close(started)
				<-release
				executed.Add(1)
				return nil
			})
			<-started
			r.Run(t.Context(), func(context.Context) error {
				executed.Add(1)
				return nil
			})
			submitted := make(chan struct{})
			go func() {
				r.Run(context.Background(), func(context.Context) error {
					executed.Add(1)
					return nil
				})
				close(submitted)
			}()
			if mode == bgtasker.RunnerTypeBlock {
				select {
				case <-submitted:
					t.Fatal("block runner dropped a task while the queue was full")
				case <-time.After(20 * time.Millisecond):
				}
			} else {
				select {
				case <-submitted:
				case <-time.After(time.Second):
					t.Fatal("drop runner blocked on a full queue")
				}
			}
			stopped := make(chan struct{})
			go func() {
				_ = r.Stop(context.Background())
				close(stopped)
			}()
			select {
			case <-submitted:
			case <-time.After(time.Second):
				t.Fatal("blocked submission did not unblock during shutdown")
			}
			releaseOnce.Do(func() { close(release) })
			select {
			case <-stopped:
			case <-time.After(time.Second):
				t.Fatal("shutdown did not finish")
			}
			require.Equal(t, int64(2), executed.Load())
		})
	}
}

func TestUnitRunnerConcurrentRunAndStop(t *testing.T) {
	for _, mode := range []string{bgtasker.RunnerTypeBlock, bgtasker.RunnerTypeDrop} {
		t.Run(mode, func(t *testing.T) {
			for range 20 {
				cfg := bgtasker.DefaultTaskConfig()
				cfg.RunnerType = mode
				r := bgtasker.NewTasker(zap.NewNop(), cfg, t.Name())
				require.NoError(t, r.Start(t.Context()))
				var wg sync.WaitGroup
				for range 8 {
					wg.Go(func() {
						for range 50 {
							r.Run(context.Background(), func(context.Context) error { return nil })
						}
					})
				}
				wg.Go(func() { _ = r.Stop(context.Background()) })
				wg.Wait()
				require.NoError(t, r.Stop(t.Context()))
			}
		})
	}
}
