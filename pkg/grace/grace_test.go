package grace_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"stg/pkg/grace"
)

type testService struct {
	start func(context.Context) error
}

func (s testService) Start(ctx context.Context) error { return s.start(ctx) }
func (testService) Stop(context.Context) error        { return nil }

func TestUnitStartStopsOnCancelledContext(t *testing.T) {
	for _, cancelBeforeStart := range []bool{true, false} {
		name := "between-services"
		if cancelBeforeStart {
			name = "before-start"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			started := 0
			if cancelBeforeStart {
				cancel()
			}
			pool := grace.NewServicePool(grace.Params{
				Logger: zap.NewNop(),
				Services: []grace.Service{
					testService{start: func(context.Context) error { started++; cancel(); return nil }},
					testService{start: func(context.Context) error { started++; return nil }},
				},
			})
			require.ErrorIs(t, pool.Start(ctx), context.Canceled)
			expected := 1
			if cancelBeforeStart {
				expected = 0
			}
			require.Equal(t, expected, started)
		})
	}
}
