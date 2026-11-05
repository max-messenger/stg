package integrations_tokenrepo_test

import (
	"context"
	"stg/internal/app/domain"
	"testing"
	"time"

	mn "github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"stg/internal/app/repository/tokenrepo"
	"stg/pkg/config"
	"stg/pkg/connections/rediscli"
	"stg/tests"
)

func TestTokenRepo(t *testing.T) {
	suite.Run(t, new(testRepo))
}

type testRepo struct {
	suite.Suite

	repo     *tokenrepo.Repository
	s3Client mn.Client
}

func (t *testRepo) SetupTest() {
	cfgFunc := config.NewConfig("test", tests.PathConfigFile)
	cfg, err := cfgFunc()
	t.NoError(err)

	redisCfg := rediscli.PoolConfig{}
	t.NoError(cfg.Provider.Get("redis").Populate(&redisCfg))

	redisPool, err := rediscli.NewPool(zap.NewNop(), redisCfg)
	t.NoError(err)

	client, err := redisPool.GetPool("main")
	t.NoError(err)

	t.NoError(client.Call(context.Background(), "ping", func(ctx context.Context, ucl redis.UniversalClient, _ rediscli.KeyFormatter) error {
		return ucl.Ping(ctx).Err()
	}))

	t.repo = tokenrepo.NewRepository(client, tokenrepo.Config{RefreshTTL: time.Second})
}

func (t *testRepo) TestStoreAndGet() {
	userID := int64(42)

	data := domain.AuthData{
		User: domain.AuthUser{
			ID: userID,
		},
	}

	err := t.repo.StoreRefresh(context.Background(), data)
	t.NoError(err)

	got, err := t.repo.GetRefresh(context.Background(), userID)
	t.NoError(err)

	t.Equal(data.User.ID, got.User.ID)
}

func (t *testRepo) TestGet_NotFound() {
	_, err := t.repo.GetRefresh(context.Background(), 99999)
	t.NoError(err)
}

func (t *testRepo) TestDelete() {
	userID := int64(43)

	data := domain.AuthData{
		User: domain.AuthUser{
			ID: userID,
		},
	}

	err := t.repo.StoreRefresh(context.Background(), data)
	t.NoError(err)

	err = t.repo.DeleteRefresh(context.Background(), userID)
	t.NoError(err)

	_, err = t.repo.GetRefresh(context.Background(), userID)
	t.NoError(err)
}
