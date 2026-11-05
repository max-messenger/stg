package rediscli_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"stg/pkg/connections/rediscli"
)

func TestUnitRedisReadOnly(t *testing.T) {
	readOnly := false
	for _, tc := range []struct {
		name string
		cfg  *bool
		want bool
	}{
		{name: "default", want: true},
		{name: "master-only", cfg: &readOnly},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := rediscli.NewRedis(zap.NewNop(), &rediscli.Config{
				Addrs: []string{"localhost:6379", "localhost:6380"}, ReadOnly: tc.cfg,
			}, t.Name())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, r.Close()) })
			require.NoError(t, r.Call(t.Context(), "options", func(
				_ context.Context, client redis.UniversalClient, _ rediscli.KeyFormatter,
			) error {
				cluster, ok := client.(*redis.ClusterClient)
				require.True(t, ok)
				require.Equal(t, tc.want, cluster.Options().ReadOnly)
				require.True(t, cluster.Options().ContextTimeoutEnabled)
				return nil
			}))
		})
	}
}

func TestUnitRedisMasterOnlyRejectsReplicaRouting(t *testing.T) {
	readOnly := false
	for _, cfg := range []*rediscli.Config{
		{ReadOnly: &readOnly, RouteByLatency: true},
		{ReadOnly: &readOnly, RouteRandomly: true},
	} {
		r, err := rediscli.NewRedis(zap.NewNop(), cfg, t.Name())
		require.Error(t, err)
		require.Nil(t, r)
	}
}
