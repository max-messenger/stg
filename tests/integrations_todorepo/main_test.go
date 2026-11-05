package integrations_todorepo

import (
	"os"
	"sync/atomic"
	"testing"

	steron "github.com/FluorescentTouch/testosteron/v2"

	"stg/internal/app/repository/todorepo"
	"stg/pkg/config"
	"stg/pkg/connections/postgres"
	"stg/tests"
)

func TestMain(m *testing.M) {
	// init package with required options
	options := []steron.Option{
		steron.NewPostgresService(tests.ImagePostgres),
	}
	cfg, err := steron.Init(options...)
	if err != nil {
		panic(err)
	}

	_ = os.Setenv(tests.EnvIntegrationPgxDns, cfg.PgConfig().Connection)

	// run the app
	code := m.Run()

	steron.Cleanup()
	os.Exit(code)
}

var userIDCounter atomic.Int64

func nextUserID() int64 {
	return userIDCounter.Add(1)
}

func prepare(t *testing.T) (r *todorepo.Repository, err error) {
	steron.ApplyMigrations(t, tests.PathMigrationDir, "")

	cfgFunc := config.NewConfig("test", tests.PathConfigFile)
	cfg, err := cfgFunc()
	if err != nil {
		return
	}

	pgCfg := postgres.PoolConfig{}
	err = cfg.Provider.Get("postgres").Populate(&pgCfg)
	if err != nil {
		return
	}

	pp := postgres.PoolParams{
		Cfgs: pgCfg,
	}

	pool, err := postgres.NewPostgresPool(pp)
	if err != nil {
		return
	}

	r, err = todorepo.NewRepository(pool)
	if err != nil {
		return
	}

	return
}
