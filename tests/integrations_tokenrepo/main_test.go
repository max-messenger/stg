package integrations_tokenrepo

import (
	"os"
	"stg/tests"
	"testing"

	steron "github.com/FluorescentTouch/testosteron/v2"
)

func TestMain(m *testing.M) {
	// init package with required options
	options := []steron.Option{
		steron.NewRedisService(tests.ImageRedis),
	}
	cfg, err := steron.Init(options...)
	if err != nil {
		panic(err)
	}

	_ = os.Setenv(tests.EnvIntegrationRedisAddr, cfg.Redis().Addr)

	// run the app
	code := m.Run()

	steron.Cleanup()
	os.Exit(code)
}
