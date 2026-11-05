package integrations_staterepo_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	mn "github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"stg/internal/app/repository/staterepo"
	"stg/pkg/config"
	"stg/pkg/connections/rediscli"
	"stg/tests"
)

func TestStateRepo(t *testing.T) {
	suite.Run(t, new(testRepo))
}

type testRepo struct {
	suite.Suite

	repo     *staterepo.Repository
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

	t.repo = staterepo.NewRepository(client)
}

func (t *testRepo) TestSetAndGet() {
	userID := int64(uuid.New().ID())

	err := t.repo.SetState(context.Background(), userID, "hello")
	t.NoError(err)

	got, err := t.repo.GetState(context.Background(), userID)
	t.NoError(err)
	t.Equal("hello", got)
}

func (t *testRepo) TestGet_Empty() {
	userID := int64(uuid.New().ID())

	got, err := t.repo.GetState(context.Background(), userID)
	t.NoError(err)
	t.Empty(got)
}

func (t *testRepo) TestGet_ConsumesValue() {
	userID := int64(uuid.New().ID())

	err := t.repo.SetState(context.Background(), userID, "once")
	t.NoError(err)

	first, err := t.repo.GetState(context.Background(), userID)
	t.NoError(err)
	t.Equal("once", first)

	second, err := t.repo.GetState(context.Background(), userID)
	t.NoError(err)
	t.Empty(second)
}

func (t *testRepo) TestClearState() {
	userID := int64(uuid.New().ID())

	err := t.repo.SetState(context.Background(), userID, "to-clear")
	t.NoError(err)

	err = t.repo.ClearState(context.Background(), userID)
	t.NoError(err)

	got, err := t.repo.GetState(context.Background(), userID)
	t.NoError(err)
	t.Empty(got)
}

func (t *testRepo) TestSet_Overwrite() {
	userID := int64(uuid.New().ID())

	err := t.repo.SetState(context.Background(), userID, "first")
	t.NoError(err)

	err = t.repo.SetState(context.Background(), userID, "second")
	t.NoError(err)

	got, err := t.repo.GetState(context.Background(), userID)
	t.NoError(err)
	t.Equal("second", got)
}
